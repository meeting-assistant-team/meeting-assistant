package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
)

// ParticipantRepository defines the interface for participant data access
type ParticipantRepository interface {
	// Create creates a new participant record
	Create(ctx context.Context, participant *entities.Participant) error

	// FindByID retrieves a participant by ID
	FindByID(ctx context.Context, id uuid.UUID) (*entities.Participant, error)

	// FindByRoomAndUser retrieves a participant by room and user ID
	FindByRoomAndUser(ctx context.Context, roomID, userID uuid.UUID) (*entities.Participant, error)

	// Update updates an existing participant
	Update(ctx context.Context, participant *entities.Participant) error

	// Delete deletes a participant record
	Delete(ctx context.Context, id uuid.UUID) error

	// FindByRoomID retrieves all participants in a room
	FindByRoomID(ctx context.Context, roomID uuid.UUID) ([]*entities.Participant, error)

	// FindActiveByRoomID retrieves all active participants in a room
	FindActiveByRoomID(ctx context.Context, roomID uuid.UUID) ([]*entities.Participant, error)

	// FindByUserID retrieves all participant records for a user
	FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entities.Participant, error)

	// CountByRoomID counts participants in a room
	CountByRoomID(ctx context.Context, roomID uuid.UUID) (int64, error)

	// CountActiveByRoomID counts active participants in a room
	CountActiveByRoomID(ctx context.Context, roomID uuid.UUID) (int64, error)

	// IsUserInRoom checks if a user is currently in a room
	IsUserInRoom(ctx context.Context, roomID, userID uuid.UUID) (bool, error)

	// FindHostByRoomID retrieves the host participant of a room
	FindHostByRoomID(ctx context.Context, roomID uuid.UUID) (*entities.Participant, error)

	// UpdateStatus updates participant status
	UpdateStatus(ctx context.Context, participantID uuid.UUID, status entities.ParticipantStatus) error

	// MarkAsJoined marks a participant as joined
	MarkAsJoined(ctx context.Context, participantID uuid.UUID) error

	// MarkAsLeft marks a participant as left
	MarkAsLeft(ctx context.Context, participantID uuid.UUID) error

	// Remove marks a participant as removed
	Remove(ctx context.Context, participantID, removedBy uuid.UUID, reason string) error

	// PromoteToHost promotes a participant to host
	PromoteToHost(ctx context.Context, participantID uuid.UUID) error

	// UpdateRole updates participant role
	UpdateRole(ctx context.Context, participantID uuid.UUID, role entities.ParticipantRole) error
}
