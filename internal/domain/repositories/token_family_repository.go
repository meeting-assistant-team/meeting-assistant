package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
)

// TokenFamilyRepository defines methods for token family persistence
type TokenFamilyRepository interface {
	Create(ctx context.Context, tokenFamily *entities.TokenFamily) error
	Update(ctx context.Context, tokenFamily *entities.TokenFamily) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*entities.TokenFamily, error)
	FindByFamilyID(ctx context.Context, familyID uuid.UUID) ([]*entities.TokenFamily, error)
	RevokeFamily(ctx context.Context, familyID uuid.UUID) error
	RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context) error
}
