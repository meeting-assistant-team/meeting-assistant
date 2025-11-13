package room

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
	"github.com/johnquangdev/meeting-assistant/internal/domain/repositories"
	usecaseErrors "github.com/johnquangdev/meeting-assistant/internal/usecase/errors"
)

// RoomService handles room business logic
type RoomService struct {
	roomRepo        repositories.RoomRepository
	participantRepo repositories.ParticipantRepository
}

// NewRoomService creates a new room service
func NewRoomService(
	roomRepo repositories.RoomRepository,
	participantRepo repositories.ParticipantRepository,
) *RoomService {
	return &RoomService{
		roomRepo:        roomRepo,
		participantRepo: participantRepo,
	}
}

// CreateRoomInput represents input for creating a room
type CreateRoomInput struct {
	Name               string
	Description        *string
	HostID             uuid.UUID
	Type               entities.RoomType
	MaxParticipants    int
	Settings           map[string]interface{}
	ScheduledStartTime *time.Time
	ScheduledEndTime   *time.Time
}

// CreateRoom creates a new room
func (s *RoomService) CreateRoom(ctx context.Context, input CreateRoomInput) (*entities.Room, error) {
	// Validate input
	if input.MaxParticipants < 2 || input.MaxParticipants > 100 {
		return nil, usecaseErrors.ErrInvalidMaxParticipants
	}

	// Generate LiveKit room name
	livekitRoomName := fmt.Sprintf("room-%s", uuid.New().String())

	// Create room entity
	room := &entities.Room{
		Name:                input.Name,
		Description:         input.Description,
		HostID:              input.HostID,
		Type:                input.Type,
		Status:              entities.RoomStatusScheduled,
		LivekitRoomName:     livekitRoomName,
		MaxParticipants:     input.MaxParticipants,
		CurrentParticipants: 0,
		ScheduledStartTime:  input.ScheduledStartTime,
		ScheduledEndTime:    input.ScheduledEndTime,
	}

	// Set default settings if not provided
	if input.Settings != nil {
		// TODO: Marshal settings to JSON
		// room.Settings = input.Settings
	}

	// Create room in database
	if err := s.roomRepo.Create(ctx, room); err != nil {
		return nil, fmt.Errorf("failed to create room: %w", err)
	}

	// Add host as first participant
	now := time.Now()
	participant := &entities.Participant{
		RoomID:         room.ID,
		UserID:         input.HostID,
		Role:           entities.ParticipantRoleHost,
		Status:         entities.ParticipantStatusInvited,
		InvitedAt:      &now,
		CanShareScreen: true,
		CanRecord:      true,
		CanMuteOthers:  true,
	}

	if err := s.participantRepo.Create(ctx, participant); err != nil {
		return nil, fmt.Errorf("failed to add host as participant: %w", err)
	}

	return room, nil
}

// GetRoom retrieves a room by ID
func (s *RoomService) GetRoom(ctx context.Context, roomID uuid.UUID) (*entities.Room, error) {
	room, err := s.roomRepo.FindByID(ctx, roomID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, usecaseErrors.ErrRoomNotFound
		}
		return nil, fmt.Errorf("failed to get room: %w", err)
	}
	return room, nil
}

// ListRooms retrieves rooms with filters
func (s *RoomService) ListRooms(ctx context.Context, filters repositories.RoomFilters) ([]*entities.Room, int64, error) {
	rooms, total, err := s.roomRepo.List(ctx, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list rooms: %w", err)
	}
	return rooms, total, nil
}

// StartRoom starts a scheduled room
func (s *RoomService) StartRoom(ctx context.Context, roomID, userID uuid.UUID) (*entities.Room, error) {
	room, err := s.roomRepo.FindByID(ctx, roomID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, usecaseErrors.ErrRoomNotFound
		}
		return nil, fmt.Errorf("failed to get room: %w", err)
	}

	// Check if user is host
	if room.HostID != userID {
		return nil, usecaseErrors.ErrNotHost
	}

	// Check if room can be started
	if room.Status == entities.RoomStatusEnded {
		return nil, usecaseErrors.ErrRoomEnded
	}

	// Start the room
	room.Start()

	if err := s.roomRepo.Update(ctx, room); err != nil {
		return nil, fmt.Errorf("failed to start room: %w", err)
	}

	return room, nil
}

// JoinRoomInput represents input for joining a room
type JoinRoomInput struct {
	RoomID uuid.UUID
	UserID uuid.UUID
}

// JoinRoom allows a user to join a room
func (s *RoomService) JoinRoom(ctx context.Context, input JoinRoomInput) (*entities.Room, *entities.Participant, error) {
	// Get room
	room, err := s.roomRepo.FindByID(ctx, input.RoomID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, usecaseErrors.ErrRoomNotFound
		}
		return nil, nil, fmt.Errorf("failed to get room: %w", err)
	}

	// Check if room has ended
	if room.IsEnded() {
		return nil, nil, usecaseErrors.ErrRoomEnded
	}

	// Check if room is full
	if room.IsFull() {
		return nil, nil, usecaseErrors.ErrRoomFull
	}

	// Check if user already in room
	isInRoom, err := s.participantRepo.IsUserInRoom(ctx, input.RoomID, input.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check user participation: %w", err)
	}
	if isInRoom {
		return nil, nil, usecaseErrors.ErrAlreadyInRoom
	}

	// Get or create participant record
	participant, err := s.participantRepo.FindByRoomAndUser(ctx, input.RoomID, input.UserID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, fmt.Errorf("failed to get participant: %w", err)
	}

	if participant == nil {
		// Create new participant
		participant = &entities.Participant{
			RoomID:         input.RoomID,
			UserID:         input.UserID,
			Role:           entities.ParticipantRoleParticipant,
			CanShareScreen: true,
		}
		if err := s.participantRepo.Create(ctx, participant); err != nil {
			return nil, nil, fmt.Errorf("failed to create participant: %w", err)
		}
	}

	// Mark as joined
	participant.Join()
	if err := s.participantRepo.Update(ctx, participant); err != nil {
		return nil, nil, fmt.Errorf("failed to update participant: %w", err)
	}

	// Increment participant count
	if err := s.roomRepo.IncrementParticipantCount(ctx, input.RoomID); err != nil {
		return nil, nil, fmt.Errorf("failed to increment participant count: %w", err)
	}

	// Start room if not started
	if room.Status == entities.RoomStatusScheduled {
		room.Start()
		if err := s.roomRepo.Update(ctx, room); err != nil {
			return nil, nil, fmt.Errorf("failed to start room: %w", err)
		}
	}

	return room, participant, nil
}

// LeaveRoom allows a user to leave a room
func (s *RoomService) LeaveRoom(ctx context.Context, roomID, userID uuid.UUID) error {
	// Get participant
	participant, err := s.participantRepo.FindByRoomAndUser(ctx, roomID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return usecaseErrors.ErrNotParticipant
		}
		return fmt.Errorf("failed to get participant: %w", err)
	}

	// Mark as left
	participant.Leave()
	if err := s.participantRepo.Update(ctx, participant); err != nil {
		return fmt.Errorf("failed to update participant: %w", err)
	}

	// Decrement participant count
	if err := s.roomRepo.DecrementParticipantCount(ctx, roomID); err != nil {
		return fmt.Errorf("failed to decrement participant count: %w", err)
	}

	// Check if room should auto-end (no active participants)
	activeCount, err := s.participantRepo.CountActiveByRoomID(ctx, roomID)
	if err != nil {
		return fmt.Errorf("failed to count active participants: %w", err)
	}

	if activeCount == 0 {
		// Auto-end the room
		if err := s.roomRepo.EndRoom(ctx, roomID); err != nil {
			return fmt.Errorf("failed to end room: %w", err)
		}
	} else if participant.IsHost() {
		// If host left, promote another participant
		if err := s.promoteNewHost(ctx, roomID); err != nil {
			return fmt.Errorf("failed to promote new host: %w", err)
		}
	}

	return nil
}

// EndRoom ends a room (host only)
func (s *RoomService) EndRoom(ctx context.Context, roomID, userID uuid.UUID) error {
	// Get room
	room, err := s.roomRepo.FindByID(ctx, roomID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return usecaseErrors.ErrRoomNotFound
		}
		return fmt.Errorf("failed to get room: %w", err)
	}

	// Check if user is host
	if room.HostID != userID {
		return usecaseErrors.ErrNotHost
	}

	// End the room
	room.End()
	if err := s.roomRepo.Update(ctx, room); err != nil {
		return fmt.Errorf("failed to end room: %w", err)
	}

	// Mark all active participants as left
	participants, err := s.participantRepo.FindActiveByRoomID(ctx, roomID)
	if err != nil {
		return fmt.Errorf("failed to get active participants: %w", err)
	}

	for _, p := range participants {
		p.Leave()
		if err := s.participantRepo.Update(ctx, p); err != nil {
			return fmt.Errorf("failed to update participant: %w", err)
		}
	}

	return nil
}

// GetParticipants retrieves all participants in a room
func (s *RoomService) GetParticipants(ctx context.Context, roomID uuid.UUID) ([]*entities.Participant, error) {
	participants, err := s.participantRepo.FindByRoomID(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}
	return participants, nil
}

// RemoveParticipant removes a participant from a room (host only)
func (s *RoomService) RemoveParticipant(ctx context.Context, roomID, hostID, participantID uuid.UUID, reason string) error {
	// Get room
	room, err := s.roomRepo.FindByID(ctx, roomID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return usecaseErrors.ErrRoomNotFound
		}
		return fmt.Errorf("failed to get room: %w", err)
	}

	// Check if user is host
	if room.HostID != hostID {
		return usecaseErrors.ErrNotHost
	}

	// Remove participant
	if err := s.participantRepo.Remove(ctx, participantID, hostID, reason); err != nil {
		return fmt.Errorf("failed to remove participant: %w", err)
	}

	// Decrement participant count
	if err := s.roomRepo.DecrementParticipantCount(ctx, roomID); err != nil {
		return fmt.Errorf("failed to decrement participant count: %w", err)
	}

	return nil
}

// TransferHost transfers host role to another participant
func (s *RoomService) TransferHost(ctx context.Context, roomID, currentHostID, newHostID uuid.UUID) error {
	// Get room
	room, err := s.roomRepo.FindByID(ctx, roomID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return usecaseErrors.ErrRoomNotFound
		}
		return fmt.Errorf("failed to get room: %w", err)
	}

	// Check if user is current host
	if room.HostID != currentHostID {
		return usecaseErrors.ErrNotHost
	}

	// Get current host participant
	currentHost, err := s.participantRepo.FindByRoomAndUser(ctx, roomID, currentHostID)
	if err != nil {
		return fmt.Errorf("failed to get current host: %w", err)
	}

	// Get new host participant
	newHost, err := s.participantRepo.FindByRoomAndUser(ctx, roomID, newHostID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return usecaseErrors.ErrNotParticipant
		}
		return fmt.Errorf("failed to get new host: %w", err)
	}

	// Demote current host to participant
	currentHost.Role = entities.ParticipantRoleParticipant
	currentHost.CanRecord = false
	currentHost.CanMuteOthers = false
	if err := s.participantRepo.Update(ctx, currentHost); err != nil {
		return fmt.Errorf("failed to demote current host: %w", err)
	}

	// Promote new host
	newHost.PromoteToHost()
	if err := s.participantRepo.Update(ctx, newHost); err != nil {
		return fmt.Errorf("failed to promote new host: %w", err)
	}

	// Update room host
	room.HostID = newHostID
	if err := s.roomRepo.Update(ctx, room); err != nil {
		return fmt.Errorf("failed to update room host: %w", err)
	}

	return nil
}

// promoteNewHost promotes the first active participant to host
func (s *RoomService) promoteNewHost(ctx context.Context, roomID uuid.UUID) error {
	participants, err := s.participantRepo.FindActiveByRoomID(ctx, roomID)
	if err != nil {
		return fmt.Errorf("failed to get active participants: %w", err)
	}

	if len(participants) == 0 {
		return nil // No participants to promote
	}

	// Promote first participant
	newHost := participants[0]
	newHost.PromoteToHost()
	if err := s.participantRepo.Update(ctx, newHost); err != nil {
		return fmt.Errorf("failed to promote participant: %w", err)
	}

	// Update room host
	room, err := s.roomRepo.FindByID(ctx, roomID)
	if err != nil {
		return fmt.Errorf("failed to get room: %w", err)
	}

	room.HostID = newHost.UserID
	if err := s.roomRepo.Update(ctx, room); err != nil {
		return fmt.Errorf("failed to update room: %w", err)
	}

	return nil
}
