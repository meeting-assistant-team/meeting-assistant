package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	// Create creates a new user
	Create(ctx context.Context, user *entities.User) error

	// FindByID finds a user by ID
	FindByID(ctx context.Context, id uuid.UUID) (*entities.User, error)

	// FindByEmail finds a user by email
	FindByEmail(ctx context.Context, email string) (*entities.User, error)

	// FindByOAuth finds a user by OAuth provider and ID
	FindByOAuth(ctx context.Context, provider, oauthID string) (*entities.User, error)

	// Update updates a user
	Update(ctx context.Context, user *entities.User) error

	// UpdateOAuthToken updates the OAuth refresh token
	UpdateOAuthToken(ctx context.Context, userID uuid.UUID, refreshToken string) error

	// UpdateLastLogin updates the last login timestamp
	UpdateLastLogin(ctx context.Context, userID uuid.UUID) error

	// Delete soft deletes a user (sets is_active to false)
	Delete(ctx context.Context, id uuid.UUID) error

	// List returns a paginated list of users
	List(ctx context.Context, limit, offset int) ([]*entities.User, error)
}
