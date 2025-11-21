package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
)

// RoomRepository defines the interface for room data access
type RoomRepository interface {
	// Create creates a new room
	Create(ctx context.Context, room *entities.Room) error

	// FindByID retrieves a room by its ID
	FindByID(ctx context.Context, id uuid.UUID) (*entities.Room, error)

	// FindBySlug retrieves a room by its slug
	FindBySlug(ctx context.Context, slug string) (*entities.Room, error)

	// FindByLivekitName retrieves a room by its LiveKit room name
	FindByLivekitName(ctx context.Context, livekitName string) (*entities.Room, error)

	// Update updates an existing room
	Update(ctx context.Context, room *entities.Room) error

	// Delete soft deletes a room
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves rooms with filters and pagination
	List(ctx context.Context, filters RoomFilters) ([]*entities.Room, int64, error)

	// FindByHostID retrieves all rooms hosted by a user
	FindByHostID(ctx context.Context, hostID uuid.UUID, limit, offset int) ([]*entities.Room, error)

	// FindActiveRooms retrieves all currently active rooms
	FindActiveRooms(ctx context.Context) ([]*entities.Room, error)

	// FindScheduledRooms retrieves all scheduled rooms
	FindScheduledRooms(ctx context.Context) ([]*entities.Room, error)

	// IncrementParticipantCount increases the participant count
	IncrementParticipantCount(ctx context.Context, roomID uuid.UUID) error

	// DecrementParticipantCount decreases the participant count
	DecrementParticipantCount(ctx context.Context, roomID uuid.UUID) error

	// UpdateStatus updates the room status
	UpdateStatus(ctx context.Context, roomID uuid.UUID, status entities.RoomStatus) error

	// EndRoom marks a room as ended and calculates duration
	EndRoom(ctx context.Context, roomID uuid.UUID) error

	// UpdateHostID updates the room's host ID
	UpdateHostID(ctx context.Context, roomID, newHostID uuid.UUID) error
}

// RoomFilters represents filter options for listing rooms
type RoomFilters struct {
	Type      *entities.RoomType
	Status    *entities.RoomStatus
	HostID    *uuid.UUID
	Search    string // Search in name, description
	Tags      []string
	Limit     int
	Offset    int
	SortBy    string // "created_at", "started_at", "name"
	SortOrder string // "asc", "desc"
}
