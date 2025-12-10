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
	CreateRoom(ctx context.Context, input CreateRoomInput) (*CreateRoomOutput, error)

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

	// GetWaitingParticipants retrieves all waiting participants in a room
	GetWaitingParticipants(ctx context.Context, roomID, hostID uuid.UUID) ([]*entities.Participant, error)

	// AdmitParticipant admits a waiting participant into the room and returns LiveKit access token
	AdmitParticipant(ctx context.Context, roomID, hostID, participantID uuid.UUID) (string, error)

	// DenyParticipant denies a waiting participant from joining the room
	DenyParticipant(ctx context.Context, roomID, hostID, participantID uuid.UUID, reason string) error

	// RemoveParticipant removes a participant from a room (host only)
	RemoveParticipant(ctx context.Context, roomID, hostID, participantID uuid.UUID, reason string) error

	// TransferHost transfers host role to another participant
	TransferHost(ctx context.Context, roomID, currentHostID, newHostID uuid.UUID) error

	// GenerateParticipantToken generates a LiveKit access token for a participant
	GenerateParticipantToken(ctx context.Context, room *entities.Room, participant *entities.Participant) (string, error)

	// GetLivekitURL returns the LiveKit server URL
	GetLivekitURL() string

	// GetRoomByLivekitName retrieves a room by LiveKit room name (for webhooks)
	GetRoomByLivekitName(ctx context.Context, livekitName string) (*entities.Room, error)

	// UpdateParticipantStatus updates participant status (for webhooks)
	UpdateParticipantStatus(ctx context.Context, roomID, userID uuid.UUID, status string) error

	// GetParticipantByRoomAndUser retrieves a participant by room and user ID
	GetParticipantByRoomAndUser(ctx context.Context, roomID, userID uuid.UUID) (*entities.Participant, error)
}

// Ensure RoomService implements Service interface
var _ Service = (*RoomService)(nil)
