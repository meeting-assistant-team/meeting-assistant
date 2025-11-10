package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
)

// SessionRepository defines the interface for session data access
type SessionRepository interface {
	// Create creates a new session
	Create(ctx context.Context, session *entities.Session) error

	// FindByID finds a session by ID
	FindByID(ctx context.Context, id uuid.UUID) (*entities.Session, error)

	// FindByTokenHash finds a session by token hash
	FindByTokenHash(ctx context.Context, tokenHash string) (*entities.Session, error)

	// FindByUserID finds all sessions for a user
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.Session, error)

	// UpdateLastUsed updates the last used timestamp
	UpdateLastUsed(ctx context.Context, sessionID uuid.UUID) error

	// Revoke revokes a session
	Revoke(ctx context.Context, sessionID uuid.UUID) error

	// RevokeAllByUserID revokes all sessions for a user
	RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error

	// DeleteExpired deletes expired sessions
	DeleteExpired(ctx context.Context, before time.Time) error

	// CleanupOldSessions removes old revoked or expired sessions
	CleanupOldSessions(ctx context.Context, before time.Time) error
}
