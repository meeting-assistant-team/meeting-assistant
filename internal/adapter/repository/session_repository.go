package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
)

// SessionRepository implements the session repository interface using GORM
type SessionRepository struct {
	db *gorm.DB
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(db *gorm.DB) *SessionRepository {
	return &SessionRepository{
		db: db,
	}
}

// Create creates a new session
func (r *SessionRepository) Create(ctx context.Context, session *entities.Session) error {
	if err := r.db.WithContext(ctx).Create(session).Error; err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

// FindByID finds a session by ID
func (r *SessionRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.Session, error) {
	var session entities.Session
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entities.ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to find session by ID: %w", err)
	}
	return &session, nil
}

// FindByTokenHash finds a session by refresh token hash
func (r *SessionRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*entities.Session, error) {
	var session entities.Session
	if err := r.db.WithContext(ctx).
		Where("refresh_token = ? AND is_revoked = ?", tokenHash, false).
		First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, entities.ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to find session by token: %w", err)
	}
	return &session, nil
}

// FindByUserID finds all sessions for a user
func (r *SessionRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Session, error) {
	var sessions []*entities.Session
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_revoked = ?", userID, false).
		Find(&sessions).Error; err != nil {
		return nil, fmt.Errorf("failed to find sessions by user ID: %w", err)
	}
	return sessions, nil
}

// Update updates a session
func (r *SessionRepository) Update(ctx context.Context, session *entities.Session) error {
	if err := r.db.WithContext(ctx).Save(session).Error; err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}
	return nil
}

// UpdateLastUsed updates the last used timestamp
func (r *SessionRepository) UpdateLastUsed(ctx context.Context, sessionID uuid.UUID) error {
	now := time.Now()
	if err := r.db.WithContext(ctx).
		Model(&entities.Session{}).
		Where("id = ?", sessionID).
		Update("last_used_at", now).Error; err != nil {
		return fmt.Errorf("failed to update last used: %w", err)
	}
	return nil
}

// Revoke revokes a session
func (r *SessionRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	if err := r.db.WithContext(ctx).
		Model(&entities.Session{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_revoked": true,
			"revoked_at": now,
		}).Error; err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}
	return nil
}

// RevokeAllByUserID revokes all sessions for a user
func (r *SessionRepository) RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	if err := r.db.WithContext(ctx).
		Model(&entities.Session{}).
		Where("user_id = ? AND is_revoked = ?", userID, false).
		Updates(map[string]interface{}{
			"is_revoked": true,
			"revoked_at": now,
		}).Error; err != nil {
		return fmt.Errorf("failed to revoke all sessions: %w", err)
	}
	return nil
}

// DeleteExpired deletes all expired sessions
func (r *SessionRepository) DeleteExpired(ctx context.Context, before time.Time) error {
	if err := r.db.WithContext(ctx).
		Where("expires_at < ?", before).
		Delete(&entities.Session{}).Error; err != nil {
		return fmt.Errorf("failed to delete expired sessions: %w", err)
	}
	return nil
}

// CleanupOldSessions removes old revoked or expired sessions
func (r *SessionRepository) CleanupOldSessions(ctx context.Context, before time.Time) error {
	if err := r.db.WithContext(ctx).
		Where("(is_revoked = ? AND revoked_at < ?) OR expires_at < ?", true, before, before).
		Delete(&entities.Session{}).Error; err != nil {
		return fmt.Errorf("failed to cleanup old sessions: %w", err)
	}
	return nil
}
