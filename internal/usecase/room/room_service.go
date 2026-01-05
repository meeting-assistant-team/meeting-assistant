package room

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"gorm.io/gorm"

	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
	"github.com/johnquangdev/meeting-assistant/internal/domain/repositories"
	lkpkg "github.com/johnquangdev/meeting-assistant/internal/infrastructure/external/livekit"
	usecaseErrors "github.com/johnquangdev/meeting-assistant/internal/usecase/errors"
	"github.com/johnquangdev/meeting-assistant/pkg/config"
)

// RoomService handles room business logic
type RoomService struct {
	roomRepo        repositories.RoomRepository
	participantRepo repositories.ParticipantRepository
	livekitClient   lkpkg.Client
	livekitURL      string
	egressClient    *lksdk.EgressClient
	storageConfig   *config.StorageConfig
	apiKey          string
	apiSecret       string
}

// NewRoomService creates a new room service
func NewRoomService(
	roomRepo repositories.RoomRepository,
	participantRepo repositories.ParticipantRepository,
	livekitClient lkpkg.Client,
	livekitURL string,
	appConfig *config.Config,
) *RoomService {
	return &RoomService{
		roomRepo:        roomRepo,
		participantRepo: participantRepo,
		livekitClient:   livekitClient,
		livekitURL:      livekitURL,
		egressClient:    lksdk.NewEgressClient(appConfig.LiveKit.URL, appConfig.LiveKit.APIKey, appConfig.LiveKit.APISecret),
		storageConfig:   &appConfig.Storage,
		apiKey:          appConfig.LiveKit.APIKey,
		apiSecret:       appConfig.LiveKit.APISecret,
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

// CreateRoomOutput represents the output of creating a room
type CreateRoomOutput struct {
	Room          *entities.Room
	LivekitToken  string
	LivekitURL    string
	LivekitRoomID string
}

// CreateRoom creates a new room
func (s *RoomService) CreateRoom(ctx context.Context, input CreateRoomInput) (*CreateRoomOutput, error) {
	// Validate input
	if input.MaxParticipants < 2 || input.MaxParticipants > 10 {
		return nil, usecaseErrors.ErrInvalidMaxParticipants
	}

	// Generate LiveKit room name
	livekitRoomName := fmt.Sprintf("room-%s", uuid.New().String())

	// Configure RoomCompositeEgress for auto-recording
	// Use public MinIO endpoint for external services to access
	publicURL := s.storageConfig.PublicURL
	if publicURL == "" {
		publicURL = fmt.Sprintf("https://%s", s.storageConfig.Endpoint)
	}

	egressConfig := &livekit.RoomEgress{
		Room: &livekit.RoomCompositeEgressRequest{
			RoomName:  livekitRoomName,
			AudioOnly: true,
			FileOutputs: []*livekit.EncodedFileOutput{
				{
					FileType: livekit.EncodedFileType_MP4,
					Filepath: "recordings/{time}-{room_name}.mp4",
					Output: &livekit.EncodedFileOutput_S3{
						S3: &livekit.S3Upload{
							AccessKey:      s.storageConfig.AccessKeyID,
							Secret:         s.storageConfig.SecretAccessKey,
							Region:         "us-east-1",
							Endpoint:       publicURL,
							Bucket:         s.storageConfig.BucketName,
							ForcePathStyle: true,
						},
					},
				},
			},
		},
	}

	// Create room in LiveKit with egress auto-recording
	roomInfo, err := s.livekitClient.CreateRoom(ctx, livekitRoomName, &lkpkg.CreateRoomOptions{
		MaxParticipants:  int32(input.MaxParticipants),
		EmptyTimeout:     300, // 5 minutes - auto-delete if no one joins
		DepartureTimeout: 30,  // 30 seconds - auto-delete after last person leaves
		Metadata:         fmt.Sprintf(`{"name":"%s","enable_recording":true}`, input.Name),
		Egress:           egressConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create livekit room: %w", err)
	}

	log.Printf("[Room] ✅ Room created with egress auto-recording enabled: %s", livekitRoomName)

	// Create room entity
	room := &entities.Room{
		Name:                input.Name,
		Description:         input.Description,
		HostID:              input.HostID,
		Type:                input.Type,
		Status:              entities.RoomStatusScheduled,
		LivekitRoomName:     livekitRoomName,
		LivekitRoomID:       &roomInfo.SID,
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
		// Cleanup: delete LiveKit room if DB insert fails
		_ = s.livekitClient.DeleteRoom(ctx, livekitRoomName)
		return nil, fmt.Errorf("failed to create room: %w", err)
	}

	// Add host as first participant
	now := time.Now()
	participant := &entities.Participant{
		RoomID:         room.ID,
		UserID:         &input.HostID,
		Role:           entities.ParticipantRoleHost,
		Status:         entities.ParticipantStatusInvited,
		InvitedAt:      &now,
		CanShareScreen: true,
		CanRecord:      true,
		CanMuteOthers:  true,
	}

	if err := s.participantRepo.Create(ctx, participant); err != nil {
		// Cleanup: delete room and LiveKit room
		_ = s.roomRepo.Delete(ctx, room.ID)
		_ = s.livekitClient.DeleteRoom(ctx, livekitRoomName)
		return nil, fmt.Errorf("failed to add host as participant: %w", err)
	}

	// Generate LiveKit access token for host
	token, err := s.livekitClient.GenerateToken(
		input.HostID.String(),
		livekitRoomName,
		"Host", // participant name
		&lkpkg.TokenOptions{
			ValidFor:       24 * time.Hour,
			CanPublish:     true,
			CanSubscribe:   true,
			CanPublishData: true,
			RoomJoin:       true,
			RoomAdmin:      true, // Host has admin rights
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate livekit token: %w", err)
	}

	return &CreateRoomOutput{
		Room:          room,
		LivekitToken:  token,
		LivekitURL:    s.livekitURL,
		LivekitRoomID: roomInfo.SID,
	}, nil
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

	// Check if room has ended (IMPORTANT: Check this BEFORE authorization)
	if room.IsEnded() {
		return nil, nil, usecaseErrors.ErrRoomEnded
	}

	// Check authorization based on room type
	if err := s.checkJoinAuthorization(ctx, room, input.UserID); err != nil {
		return nil, nil, err
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

	// If participant exists, check if they are blocked or removed
	if participant != nil {
		// Check if user is blocked (denied status) or has been removed
		if participant.Status == entities.ParticipantStatusDenied || participant.IsRemoved {
			return nil, nil, fmt.Errorf("you have been blocked from this room")
		}

		// Check if user was removed (without being blocked)
		if participant.Status == entities.ParticipantStatusRemoved {
			return nil, nil, fmt.Errorf("you have been removed from this room")
		}

		// If participant already joined, return error
		if participant.Status == entities.ParticipantStatusJoined {
			return nil, nil, usecaseErrors.ErrAlreadyInRoom
		}

		// Allow rejoin only for: left, invited, or waiting status
		if participant.Status == entities.ParticipantStatusLeft ||
			participant.Status == entities.ParticipantStatusInvited ||
			participant.Status == entities.ParticipantStatusWaiting {
			// Update status based on role and room type
			if room.HostID == input.UserID {
				// Host can join immediately
				participant.Status = entities.ParticipantStatusJoined
			} else {
				// Regular users go to waiting room (will be updated by authorization check)
				participant.Status = entities.ParticipantStatusWaiting
			}
			if err := s.participantRepo.Update(ctx, participant); err != nil {
				return nil, nil, fmt.Errorf("failed to update participant: %w", err)
			}
		} else {
			// Invalid status for rejoining
			return nil, nil, fmt.Errorf("cannot rejoin room with current status: %s", participant.Status)
		}
	} else {
		// Create new participant
		participantRole := entities.ParticipantRoleParticipant
		participantStatus := entities.ParticipantStatusWaiting
		if room.HostID == input.UserID {
			participantRole = entities.ParticipantRoleHost
			participantStatus = entities.ParticipantStatusJoined
		}
		participant = &entities.Participant{
			RoomID:         input.RoomID,
			UserID:         &input.UserID,
			Role:           participantRole,
			Status:         participantStatus,
			CanShareScreen: true,
		}
		if err := s.participantRepo.Create(ctx, participant); err != nil {
			return nil, nil, fmt.Errorf("failed to create participant: %w", err)
		}
	}

	// Nếu là host, cho join luôn
	if room.HostID == input.UserID {
		// Increment participant count nếu vừa tạo mới
		if participant.Status == entities.ParticipantStatusJoined {
			if err := s.roomRepo.IncrementParticipantCount(ctx, input.RoomID); err != nil {
				return nil, nil, fmt.Errorf("failed to increment participant count: %w", err)
			}
		}
		// Start room nếu chưa bắt đầu
		if room.Status == entities.RoomStatusScheduled {
			room.Start()
			if err := s.roomRepo.Update(ctx, room); err != nil {
				return nil, nil, fmt.Errorf("failed to start room: %w", err)
			}
		}
		return room, participant, nil
	}

	// Nếu không phải host, return participant với status waiting (không throw error)
	// Handler sẽ check status và return 200 với message chờ duyệt
	return room, participant, nil
}

// checkJoinAuthorization checks if a user is authorized to join a room
func (s *RoomService) checkJoinAuthorization(ctx context.Context, room *entities.Room, userID uuid.UUID) error {
	// Host can always join their own room
	if room.HostID == userID {
		return nil
	}

	switch room.Type {
	case entities.RoomTypePublic:
		// Anyone can join public rooms
		return nil

	case entities.RoomTypePrivate:
		// Must be invited to join private rooms
		participant, err := s.participantRepo.FindByRoomAndUser(ctx, room.ID, userID)
		if err != nil || participant == nil {
			return usecaseErrors.ErrNotInvited
		}

		// Check if invited (not yet joined or left)
		if participant.Status != entities.ParticipantStatusInvited {
			// If already joined, return specific error
			if participant.Status == entities.ParticipantStatusJoined {
				return usecaseErrors.ErrAlreadyInRoom
			}
			// If declined or removed, deny access
			if participant.Status == entities.ParticipantStatusDeclined ||
				participant.Status == entities.ParticipantStatusRemoved {
				return usecaseErrors.ErrAccessDenied
			}
			return usecaseErrors.ErrNotInvited
		}

		return nil

	case entities.RoomTypeScheduled:
		// Must be invited to join scheduled rooms
		participant, err := s.participantRepo.FindByRoomAndUser(ctx, room.ID, userID)
		if err != nil || participant == nil {
			return usecaseErrors.ErrNotInvited
		}

		// Check if invited
		if participant.Status != entities.ParticipantStatusInvited {
			if participant.Status == entities.ParticipantStatusJoined {
				return usecaseErrors.ErrAlreadyInRoom
			}
			return usecaseErrors.ErrNotInvited
		}

		// Check time window (allow join 15 mins before scheduled time)
		if room.ScheduledStartTime != nil {
			now := time.Now()
			allowedTime := room.ScheduledStartTime.Add(-15 * time.Minute)
			if now.Before(allowedTime) {
				return usecaseErrors.ErrTooEarly
			}
		}

		return nil

	default:
		return fmt.Errorf("unknown room type: %s", room.Type)
	}
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

	// Check if participant was actually joined (not just waiting)
	wasJoined := participant.Status == entities.ParticipantStatusJoined

	// Mark as left
	participant.Leave()
	if err := s.participantRepo.Update(ctx, participant); err != nil {
		return fmt.Errorf("failed to update participant: %w", err)
	}

	// Only decrement count if participant was actually joined (not waiting/denied/etc)
	if wasJoined {
		if err := s.roomRepo.DecrementParticipantCount(ctx, roomID); err != nil {
			return fmt.Errorf("failed to decrement participant count: %w", err)
		}
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
	} else if wasJoined && participant.IsHost() {
		// If host left, promote another participant (only if host was actually joined)
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

	// Get all active participants to remove them from LiveKit
	participants, err := s.participantRepo.FindActiveByRoomID(ctx, roomID)
	if err != nil {
		return fmt.Errorf("failed to get active participants: %w", err)
	}

	// Remove all participants from LiveKit room (kick them out)
	for _, p := range participants {
		if err := s.livekitClient.RemoveParticipant(ctx, room.LivekitRoomName, p.UserID.String()); err != nil {
			// Log error but continue with other participants
			fmt.Printf("⚠️  warning: failed to remove participant %s from livekit: %v\n", p.UserID.String(), err)
		}
	}

	// Delete room from LiveKit (closes room and ensures it's removed)
	if err := s.livekitClient.DeleteRoom(ctx, room.LivekitRoomName); err != nil {
		// Log error but don't fail - room status should still be updated in DB
		fmt.Printf("⚠️  warning: failed to delete livekit room %s: %v\n", room.LivekitRoomName, err)
	}

	// End the room in database
	room.End()
	if err := s.roomRepo.Update(ctx, room); err != nil {
		return fmt.Errorf("failed to end room: %w", err)
	}

	// Mark all active participants as left
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

// GetWaitingParticipants retrieves all waiting participants in a room
func (s *RoomService) GetWaitingParticipants(ctx context.Context, roomID, hostID uuid.UUID) ([]*entities.Participant, error) {
	// Verify room exists
	room, err := s.roomRepo.FindByID(ctx, roomID)
	if err != nil {
		return nil, usecaseErrors.ErrRoomNotFound
	}

	// Check if room has ended
	if room.IsEnded() {
		return nil, usecaseErrors.ErrRoomEnded
	}

	// Verify user is the host
	if room.HostID != hostID {
		return nil, usecaseErrors.ErrNotHost
	}

	// Get waiting participants
	participants, err := s.participantRepo.FindWaitingByRoomID(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get waiting participants: %w", err)
	}

	return participants, nil
}

// AdmitParticipant admits a waiting participant into the room and generates LiveKit access token
func (s *RoomService) AdmitParticipant(ctx context.Context, roomID, hostID, participantID uuid.UUID) (string, error) {
	// Verify room exists
	room, err := s.roomRepo.FindByID(ctx, roomID)
	if err != nil {
		return "", usecaseErrors.ErrRoomNotFound
	}

	// Check if room has ended
	if room.IsEnded() {
		return "", usecaseErrors.ErrRoomEnded
	}

	// Verify user is the host
	if room.HostID != hostID {
		return "", usecaseErrors.ErrNotHost
	}

	// Get participant
	participant, err := s.participantRepo.FindByID(ctx, participantID)
	if err != nil {
		return "", usecaseErrors.ErrParticipantNotFound
	}

	// Verify participant is waiting
	if participant.Status != entities.ParticipantStatusWaiting {
		return "", usecaseErrors.ErrInvalidParticipantStatus
	}

	// Mark as joined
	if err := s.participantRepo.MarkAsJoined(ctx, participantID); err != nil {
		return "", fmt.Errorf("failed to admit participant: %w", err)
	}

	// Increment participant count
	if err := s.roomRepo.IncrementParticipantCount(ctx, roomID); err != nil {
		return "", fmt.Errorf("failed to increment participant count: %w", err)
	}

	// Generate LiveKit access token for the participant
	accessToken, err := s.livekitClient.GenerateToken(participant.UserID.String(), room.LivekitRoomName, "", nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate access token: %w", err)
	}

	return accessToken, nil
}

// DenyParticipant denies a waiting participant from joining the room
// This is a soft rejection - the participant record is deleted so the user can try to join again
// For permanent blocking, use BlockParticipant instead
func (s *RoomService) DenyParticipant(ctx context.Context, roomID, hostID, participantID uuid.UUID, reason string) error {
	// Verify room exists
	room, err := s.roomRepo.FindByID(ctx, roomID)
	if err != nil {
		return usecaseErrors.ErrRoomNotFound
	}

	// Check if room has ended
	if room.IsEnded() {
		return usecaseErrors.ErrRoomEnded
	}

	// Verify user is the host
	if room.HostID != hostID {
		return usecaseErrors.ErrNotHost
	}

	// Get participant
	participant, err := s.participantRepo.FindByID(ctx, participantID)
	if err != nil {
		return usecaseErrors.ErrParticipantNotFound
	}

	// Verify participant is waiting
	if participant.Status != entities.ParticipantStatusWaiting {
		return usecaseErrors.ErrInvalidParticipantStatus
	}

	// Delete the participant record instead of marking as denied
	// This allows the user to try joining again later
	if err := s.participantRepo.Delete(ctx, participantID); err != nil {
		return fmt.Errorf("failed to deny participant: %w", err)
	}

	return nil
}

// BlockParticipant permanently blocks a participant from joining the room
// Unlike DenyParticipant, this creates a permanent block record
func (s *RoomService) BlockParticipant(ctx context.Context, roomID, hostID, participantID uuid.UUID, reason string) error {
	// Verify room exists
	room, err := s.roomRepo.FindByID(ctx, roomID)
	if err != nil {
		return usecaseErrors.ErrRoomNotFound
	}

	// Check if room has ended
	if room.IsEnded() {
		return usecaseErrors.ErrRoomEnded
	}

	// Verify user is the host
	if room.HostID != hostID {
		return usecaseErrors.ErrNotHost
	}

	// Get participant
	participant, err := s.participantRepo.FindByID(ctx, participantID)
	if err != nil {
		return usecaseErrors.ErrParticipantNotFound
	}

	// Cannot block the host
	if participant.UserID != nil && *participant.UserID == room.HostID {
		return fmt.Errorf("cannot block the host")
	}

	// Set status to denied (permanent block)
	// Also mark as removed with reason
	removalReason := "Blocked by host"
	if reason != "" {
		removalReason = fmt.Sprintf("Blocked: %s", reason)
	}
	participant.Status = entities.ParticipantStatusDenied
	participant.IsRemoved = true
	participant.RemovedBy = &hostID
	participant.RemovalReason = &removalReason

	if err := s.participantRepo.Update(ctx, participant); err != nil {
		return fmt.Errorf("failed to block participant: %w", err)
	}

	return nil
}

// GetMyParticipantStatus gets the current user's participant status in a room
// This is used for polling - user checks their status and gets token if admitted
func (s *RoomService) GetMyParticipantStatus(ctx context.Context, roomID, userID uuid.UUID) (*entities.Room, *entities.Participant, string, error) {
	// Get room
	room, err := s.roomRepo.FindByID(ctx, roomID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, "", usecaseErrors.ErrRoomNotFound
		}
		return nil, nil, "", fmt.Errorf("failed to get room: %w", err)
	}

	// Get participant record
	participant, err := s.participantRepo.FindByRoomAndUser(ctx, roomID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, "", usecaseErrors.ErrParticipantNotFound
		}
		return nil, nil, "", fmt.Errorf("failed to get participant: %w", err)
	}

	// If participant is joined, generate token
	var token string
	if participant.Status == entities.ParticipantStatusJoined {
		token, err = s.GenerateParticipantToken(ctx, room, participant)
		if err != nil {
			return nil, nil, "", fmt.Errorf("failed to generate token: %w", err)
		}
	}

	return room, participant, token, nil
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

	// Không cho host tự remove chính mình
	if participantID == hostID {
		return usecaseErrors.ErrCannotRemoveSelf
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

	// Không cho transfer cho chính mình
	if currentHostID == newHostID {
		return usecaseErrors.ErrCannotTransferToSelf
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

	// Không cho transfer cho người đã rời phòng (status phải là joined)
	if newHost.Status != entities.ParticipantStatusJoined {
		return usecaseErrors.ErrInvalidParticipantStatus
	}

	// Demote current host to participant
	currentHost.Role = entities.ParticipantRoleParticipant
	currentHost.CanRecord = false
	currentHost.CanMuteOthers = false
	if err := s.participantRepo.Update(ctx, currentHost); err != nil {
		return fmt.Errorf("failed to demote current host: %w", err)
	}

	// Promote new host (đảm bảo chỉ có 1 host)
	newHost.PromoteToHost()
	if err := s.participantRepo.Update(ctx, newHost); err != nil {
		return fmt.Errorf("failed to promote new host: %w", err)
	}

	// Update room host - Use dedicated UpdateHostID method to ensure update
	if err := s.roomRepo.UpdateHostID(ctx, roomID, newHostID); err != nil {
		return fmt.Errorf("failed to update room host: %w", err)
	}

	// DEBUG: Verify update by re-fetching
	verifyRoom, err := s.roomRepo.FindByID(ctx, roomID)
	if err != nil {
		fmt.Printf("⚠️  [TRANSFER_HOST] Failed to verify update: %v\n", err)
	} else {
		fmt.Printf("🔍 [TRANSFER_HOST] Verification - Room %s HostID after update: %s (expected: %s)\n",
			roomID, verifyRoom.HostID, newHostID)
	}

	// DEBUG: Confirm transfer
	fmt.Printf("✅ [TRANSFER_HOST] Room %s - Host transferred from %s to %s\n",
		roomID, currentHostID, newHostID)

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

	room.HostID = *newHost.UserID
	if err := s.roomRepo.Update(ctx, room); err != nil {
		return fmt.Errorf("failed to update room: %w", err)
	}

	return nil
}

// GenerateParticipantToken generates a LiveKit access token for a participant
func (s *RoomService) GenerateParticipantToken(ctx context.Context, room *entities.Room, participant *entities.Participant) (string, error) {
	// Determine participant name and admin rights
	participantName := "Participant"
	isAdmin := participant.IsHost()

	if participant.IsHost() {
		participantName = "Host"
	}

	// Generate token
	token, err := s.livekitClient.GenerateToken(
		participant.UserID.String(),
		room.LivekitRoomName,
		participantName,
		&lkpkg.TokenOptions{
			ValidFor:       24 * time.Hour,
			CanPublish:     true,
			CanSubscribe:   true,
			CanPublishData: true,
			RoomJoin:       true,
			RoomAdmin:      isAdmin,
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate participant token: %w", err)
	}

	return token, nil
}

// GetLivekitURL returns the LiveKit server URL
func (s *RoomService) GetLivekitURL() string {
	return s.livekitURL
}

// GetRoomByLivekitName retrieves a room by its LiveKit room name
func (s *RoomService) GetRoomByLivekitName(ctx context.Context, livekitName string) (*entities.Room, error) {
	room, err := s.roomRepo.FindByLivekitName(ctx, livekitName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, usecaseErrors.ErrRoomNotFound
		}
		return nil, fmt.Errorf("failed to get room by livekit name: %w", err)
	}
	return room, nil
}

// UpdateParticipantStatus updates participant status (used by webhooks)
func (s *RoomService) UpdateParticipantStatus(ctx context.Context, roomID, userID uuid.UUID, status string) error {
	participant, err := s.participantRepo.FindByRoomAndUser(ctx, roomID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return usecaseErrors.ErrNotParticipant
		}
		return fmt.Errorf("failed to get participant: %w", err)
	}

	// Update status based on string value
	switch status {
	case "joined":
		if participant.Status != entities.ParticipantStatusJoined {
			participant.Join()
			if err := s.participantRepo.Update(ctx, participant); err != nil {
				return fmt.Errorf("failed to update participant status: %w", err)
			}
		}
	case "left":
		participant.Leave()
		if err := s.participantRepo.Update(ctx, participant); err != nil {
			return fmt.Errorf("failed to update participant status: %w", err)
		}
	}

	return nil
}

// GetParticipantByRoomAndUser retrieves a participant by room and user ID
func (s *RoomService) GetParticipantByRoomAndUser(ctx context.Context, roomID, userID uuid.UUID) (*entities.Participant, error) {
	return s.participantRepo.FindByRoomAndUser(ctx, roomID, userID)
}

// InviteUserByEmail invites a user to join a room by email
func (s *RoomService) InviteUserByEmail(ctx context.Context, roomID, inviterID uuid.UUID, email string) (*entities.Participant, error) {
	// Verify room exists
	room, err := s.roomRepo.FindByID(ctx, roomID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, usecaseErrors.ErrRoomNotFound
		}
		return nil, fmt.Errorf("failed to get room: %w", err)
	}

	// Verify inviter is host
	inviter, err := s.participantRepo.FindByRoomAndUser(ctx, roomID, inviterID)
	if err != nil {
		return nil, usecaseErrors.ErrNotParticipant
	}
	if !inviter.IsHost() {
		return nil, usecaseErrors.ErrNotHost
	}

	// Check if already invited/participating with this email
	existing, err := s.participantRepo.FindByRoomAndEmail(ctx, roomID, email)
	now := time.Now()

	// If invitation already exists, handle based on status
	if err == nil && existing != nil {
		switch existing.Status {
		case entities.ParticipantStatusInvited:
			// Already pending - return existing invitation (idempotent for network lag)
			log.Printf("[Invitation] User already invited (pending): email=%s, room=%s", email, room.Name)
			return existing, nil

		case entities.ParticipantStatusJoined:
			// Already in room - no need to invite again
			return nil, usecaseErrors.ErrAlreadyInvited

		case entities.ParticipantStatusDeclined, entities.ParticipantStatusLeft:
			// User declined or left - allow re-invitation by updating status
			existing.Status = entities.ParticipantStatusInvited
			existing.InvitedBy = &inviterID
			existing.InvitedAt = &now
			if err := s.participantRepo.Update(ctx, existing); err != nil {
				return nil, fmt.Errorf("failed to update invitation: %w", err)
			}
			log.Printf("[Invitation] User re-invited to room: email=%s, room=%s, previous_status=%s",
				email, room.Name, existing.Status)
			return existing, nil
		}
	}

	// Create new invitation participant record
	participant := &entities.Participant{
		RoomID:       roomID,
		UserID:       nil, // Not assigned yet
		Role:         entities.ParticipantRoleParticipant,
		Status:       entities.ParticipantStatusInvited,
		InvitedEmail: &email,
		InvitedBy:    &inviterID,
		InvitedAt:    &now,
	}

	if err := s.participantRepo.Create(ctx, participant); err != nil {
		return nil, fmt.Errorf("failed to create invitation: %w", err)
	}

	log.Printf("[Invitation] User invited to room: email=%s, room=%s, inviter=%s", email, room.Name, inviterID)

	return participant, nil
}

// GetInvitationsByEmail retrieves all invitations for a given email
func (s *RoomService) GetInvitationsByEmail(ctx context.Context, email string) ([]*entities.Participant, error) {
	participants, err := s.participantRepo.FindByInvitedEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get invitations: %w", err)
	}
	return participants, nil
}

// AcceptInvitationByEmail accepts an invitation and joins the room
func (s *RoomService) AcceptInvitationByEmail(ctx context.Context, roomID uuid.UUID, email string, userID uuid.UUID) (*entities.Room, *entities.Participant, string, error) {
	// Find invitation
	participant, err := s.participantRepo.FindByRoomAndEmail(ctx, roomID, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, "", usecaseErrors.ErrInvitationNotFound
		}
		return nil, nil, "", fmt.Errorf("failed to find invitation: %w", err)
	}

	// Verify room exists and is not ended
	room, err := s.roomRepo.FindByID(ctx, roomID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, "", usecaseErrors.ErrRoomNotFound
		}
		return nil, nil, "", fmt.Errorf("failed to get room: %w", err)
	}

	if room.Status == entities.RoomStatusEnded || room.Status == entities.RoomStatusCancelled {
		return nil, nil, "", usecaseErrors.ErrRoomEnded
	}

	// Update participant record with user ID and join
	participant.UserID = &userID
	participant.Status = entities.ParticipantStatusJoined
	now := time.Now()
	participant.JoinedAt = &now

	if err := s.participantRepo.Update(ctx, participant); err != nil {
		return nil, nil, "", fmt.Errorf("failed to update participant: %w", err)
	}

	// Update room participant count
	if room.Status == entities.RoomStatusActive {
		room.CurrentParticipants++
		if err := s.roomRepo.Update(ctx, room); err != nil {
			return nil, nil, "", fmt.Errorf("failed to update room: %w", err)
		}
	}

	// Generate LiveKit token
	token, err := s.GenerateParticipantToken(ctx, room, participant)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	log.Printf("[Invitation] User accepted invitation: email=%s, room=%s, user=%s", email, room.Name, userID)

	return room, participant, token, nil
}

// DeclineInvitationByEmail declines an invitation
func (s *RoomService) DeclineInvitationByEmail(ctx context.Context, roomID uuid.UUID, email string, userID uuid.UUID) error {
	// Find invitation
	participant, err := s.participantRepo.FindByRoomAndEmail(ctx, roomID, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return usecaseErrors.ErrInvitationNotFound
		}
		return fmt.Errorf("failed to find invitation: %w", err)
	}

	// Update status to declined
	participant.Status = entities.ParticipantStatusDeclined
	participant.UserID = &userID // Link to user who declined

	if err := s.participantRepo.Update(ctx, participant); err != nil {
		return fmt.Errorf("failed to update participant: %w", err)
	}

	log.Printf("[Invitation] User declined invitation: email=%s, room_id=%s, user=%s", email, roomID, userID)

	return nil
}

// GetRoomInvitations retrieves all invitations for a room (host only)
func (s *RoomService) GetRoomInvitations(ctx context.Context, roomID, hostID uuid.UUID) ([]*entities.Participant, error) {
	// Verify host permission
	host, err := s.participantRepo.FindByRoomAndUser(ctx, roomID, hostID)
	if err != nil {
		return nil, usecaseErrors.ErrNotParticipant
	}
	if !host.IsHost() {
		return nil, usecaseErrors.ErrNotHost
	}

	// Get all invitations for the room
	invitations, err := s.participantRepo.FindInvitedByRoomID(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get invitations: %w", err)
	}

	return invitations, nil
}
