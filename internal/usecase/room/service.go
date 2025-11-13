package room

import (
	"context"

	"github.com/google/uuid"
	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
	"github.com/johnquangdev/meeting-assistant/internal/domain/repositories"
)

// Service defines the interface for room use case
type Service interface {
	// CreateRoom creates a new room
	CreateRoom(ctx context.Context, input CreateRoomInput) (*entities.Room, error)

	// GetRoom retrieves a room by ID
	GetRoom(ctx context.Context, roomID uuid.UUID) (*entities.Room, error)

	// ListRooms retrieves rooms with filters
	ListRooms(ctx context.Context, filters repositories.RoomFilters) ([]*entities.Room, int64, error)

	// StartRoom starts a scheduled room
	StartRoom(ctx context.Context, roomID, userID uuid.UUID) (*entities.Room, error)

	// JoinRoom allows a user to join a room
	JoinRoom(ctx context.Context, input JoinRoomInput) (*entities.Room, *entities.Participant, error)

	// LeaveRoom allows a user to leave a room
	LeaveRoom(ctx context.Context, roomID, userID uuid.UUID) error

	// EndRoom ends a room (host only)
	EndRoom(ctx context.Context, roomID, userID uuid.UUID) error

	// GetParticipants retrieves all participants in a room
	GetParticipants(ctx context.Context, roomID uuid.UUID) ([]*entities.Participant, error)

	// RemoveParticipant removes a participant from a room (host only)
	RemoveParticipant(ctx context.Context, roomID, hostID, participantID uuid.UUID, reason string) error

	// TransferHost transfers host role to another participant
	TransferHost(ctx context.Context, roomID, currentHostID, newHostID uuid.UUID) error
}

// Ensure RoomService implements Service interface
var _ Service = (*RoomService)(nil)
