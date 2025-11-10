package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
)

// UserRepository implements the user repository interface using GORM
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *entities.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// FindByID finds a user by ID
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	var user entities.User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entities.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by ID: %w", err)
	}
	return &user, nil
}

// FindByEmail finds a user by email
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*entities.User, error) {
	var user entities.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entities.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}
	return &user, nil
}

// FindByOAuth finds a user by OAuth provider and ID
func (r *UserRepository) FindByOAuth(ctx context.Context, provider, oauthID string) (*entities.User, error) {
	var user entities.User
	if err := r.db.WithContext(ctx).
		Where("oauth_provider = ? AND oauth_id = ?", provider, oauthID).
		First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entities.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by OAuth: %w", err)
	}
	return &user, nil
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, user *entities.User) error {
	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// UpdateOAuthToken updates the OAuth refresh token
func (r *UserRepository) UpdateOAuthToken(ctx context.Context, userID uuid.UUID, refreshToken string) error {
	if err := r.db.WithContext(ctx).
		Model(&entities.User{}).
		Where("id = ?", userID).
		Update("oauth_refresh_token", refreshToken).Error; err != nil {
		return fmt.Errorf("failed to update OAuth token: %w", err)
	}
	return nil
}

// UpdateLastLogin updates the last login timestamp
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	if err := r.db.WithContext(ctx).
		Model(&entities.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"last_login_at":  now,
			"last_active_at": now,
			"updated_at":     now,
		}).Error; err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}
	return nil
}

// Delete deletes a user (soft delete)
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&entities.User{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// List lists users with pagination
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*entities.User, error) {
	var users []*entities.User
	if err := r.db.WithContext(ctx).
		Limit(limit).
		Offset(offset).
		Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return users, nil
}
