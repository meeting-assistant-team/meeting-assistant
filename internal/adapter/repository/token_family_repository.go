package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
)

type tokenFamilyRepository struct {
	db *gorm.DB
}

// NewTokenFamilyRepository creates a new token family repository
func NewTokenFamilyRepository(db *gorm.DB) *tokenFamilyRepository {
	return &tokenFamilyRepository{db: db}
}

// Create creates a new token family
func (r *tokenFamilyRepository) Create(ctx context.Context, tokenFamily *entities.TokenFamily) error {
	return r.db.WithContext(ctx).Create(tokenFamily).Error
}

// Update updates an existing token family
func (r *tokenFamilyRepository) Update(ctx context.Context, tokenFamily *entities.TokenFamily) error {
	return r.db.WithContext(ctx).Save(tokenFamily).Error
}

// FindByTokenHash finds a token family by refresh token hash
// Uses SELECT FOR UPDATE to lock the row and prevent race conditions during token rotation
func (r *tokenFamilyRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*entities.TokenFamily, error) {
	var tokenFamily entities.TokenFamily
	err := r.db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("refresh_token_hash = ?", tokenHash).
		First(&tokenFamily).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entities.ErrTokenFamilyNotFound
		}
		return nil, fmt.Errorf("failed to find token family: %w", err)
	}

	return &tokenFamily, nil
}

// FindByFamilyID finds all tokens in a family chain
func (r *tokenFamilyRepository) FindByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*entities.TokenFamily, error) {
	var tokenFamilies []*entities.TokenFamily
	err := r.db.WithContext(ctx).
		Where("family_id = ?", familyID).
		Order("created_at DESC").
		Find(&tokenFamilies).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find token families: %w", err)
	}

	return tokenFamilies, nil
}

// RevokeFamily revokes all tokens in a family (used when theft detected)
func (r *tokenFamilyRepository) RevokeFamily(ctx context.Context, familyID uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&entities.TokenFamily{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", now).Error
}

// RevokeAllByUserID revokes all token families for a user
func (r *tokenFamilyRepository) RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&entities.TokenFamily{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}

// DeleteExpired deletes expired token families (cleanup job)
func (r *tokenFamilyRepository) DeleteExpired(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&entities.TokenFamily{}).Error
}
