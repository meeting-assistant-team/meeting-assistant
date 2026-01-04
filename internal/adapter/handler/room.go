package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/johnquangdev/meeting-assistant/errors"
	summaryDTO "github.com/johnquangdev/meeting-assistant/internal/adapter/dto"
	"github.com/johnquangdev/meeting-assistant/internal/adapter/dto/room"
	"github.com/johnquangdev/meeting-assistant/internal/adapter/presenter"
	"github.com/johnquangdev/meeting-assistant/internal/adapter/repository"
	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
	domainrepo "github.com/johnquangdev/meeting-assistant/internal/domain/repositories"
	roomUsecase "github.com/johnquangdev/meeting-assistant/internal/usecase/room"
)

// Room handles room-related HTTP requests
type Room struct {
	roomService roomUsecase.Service
	aiJobRepo   *repository.AIJobRepository
	summaryRepo domainrepo.AIRepository
	logger      *zap.Logger
}

// NewRoomHandler creates a new room handler
func NewRoomHandler(roomService roomUsecase.Service, aiJobRepo *repository.AIJobRepository, summaryRepo domainrepo.AIRepository, logger *zap.Logger) *Room {
	return &Room{
		roomService: roomService,
		aiJobRepo:   aiJobRepo,
		summaryRepo: summaryRepo,
		logger:      logger,
	}
}

// CreateRoom handles POST /rooms
// @Summary      Create a new room
// @Description  Creates a new meeting room with specified settings
// @Tags         Rooms
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      room.CreateRoomRequest  true  "Room creation request"
// @Success      201      {object}  room.RoomResponse  "Room created successfully"
// @Failure      400      {object}  map[string]interface{}  "Invalid request or validation failed"
// @Failure      401      {object}  map[string]interface{}  "User not authenticated"
// @Failure      500      {object}  map[string]interface{}  "Failed to create room"
// @Router       /rooms [post]
func (h *Room) CreateRoom(c echo.Context) error {
	var req room.CreateRoomRequest
	if err := c.Bind(&req); err != nil {
		return h.handleError(c, errors.ErrInvalidArgument("Invalid request body"))
	}

	// Validate request
	if err := c.Validate(&req); err != nil {
		return h.handleError(c, errors.ErrInvalidArgument("Validation failed").WithDetail("error", err.Error()))
	}

	// Get user ID from context (set by auth middleware)
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return h.handleError(c, errors.ErrUnauthenticated().WithDetail("error", "User not authenticated"))
	}

	// Parse room type
	var roomType entities.RoomType
	switch req.Type {
	case "public":
		roomType = entities.RoomTypePublic
	case "private":
		roomType = entities.RoomTypePrivate
	case "scheduled":
		roomType = entities.RoomTypeScheduled
	default:
		return h.handleError(c, errors.ErrInvalidArgument("Invalid room type").WithDetail("error", "Room type must be public, private, or scheduled"))
	}

	// Create room
	input := roomUsecase.CreateRoomInput{
		Name:               req.Name,
		Description:        req.Description,
		HostID:             userID,
		Type:               roomType,
		MaxParticipants:    req.MaxParticipants,
		Settings:           req.Settings,
		ScheduledStartTime: req.ScheduledStartTime,
		ScheduledEndTime:   req.ScheduledEndTime,
	}

	output, err := h.roomService.CreateRoom(c.Request().Context(), input)
	if err != nil {
		return h.handleError(c, errors.ErrInternal(err))
	}

	response := &room.CreateRoomResponse{
		Room:         presenter.ToRoomResponse(output.Room),
		LivekitToken: output.LivekitToken,
		LivekitURL:   output.LivekitURL,
	}

	return h.handleSuccess(c, response)
}

// GetRoom handles GET /rooms/:id
// @Summary      Get room details
// @Description  Gets detailed information about a specific room
// @Tags         Rooms
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Room ID (UUID)"
// @Success      200  {object}  room.RoomResponse  "Room details"
// @Failure      400  {object}  map[string]interface{}  "Invalid room ID"
// @Failure      404  {object}  map[string]interface{}  "Room not found"
// @Router       /rooms/{id} [get]
func (h *Room) GetRoom(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return h.handleError(c, errors.ErrInvalidArgument("Invalid room ID").WithDetail("error", "Room ID must be a valid UUID"))
	}

	r, err := h.roomService.GetRoom(c.Request().Context(), roomID)
	if err != nil {
		return h.handleError(c, errors.ErrNotFound("Room not found").WithDetail("error", err.Error()))
	}

	return h.handleSuccess(c, presenter.ToRoomResponse(r))
}

// ListRooms handles GET /rooms
// @Summary      List rooms
// @Description  Gets a paginated list of rooms with optional filters
// @Tags         Rooms
// @Produce      json
// @Security     BearerAuth
// @Param        page       query     int     false  "Page number (default: 1)"
// @Param        page_size  query     int     false  "Items per page (default: 20)"
// @Param        type       query     string  false  "Room type filter (public/private/scheduled)"
// @Param        status     query     string  false  "Room status filter (scheduled/active/ended/cancelled)"
// @Param        search     query     string  false  "Search by room name"
// @Param        tags       query     array   false  "Filter by tags"
// @Param        sort_by    query     string  false  "Sort field (created_at/start_time/participant_count)"
// @Param        sort_order query     string  false  "Sort order (asc/desc)"
// @Success      200        {object}  room.RoomListResponse  "List of rooms"
// @Failure      400        {object}  map[string]interface{}  "Invalid request"
// @Failure      500        {object}  map[string]interface{}  "Failed to list rooms"
// @Router       /rooms [get]
func (h *Room) ListRooms(c echo.Context) error {
	// Manual binding for query parameters to avoid Echo bind conflicts
	var req room.ListRoomsRequest

	// Parse query params directly
	req.Type = c.QueryParam("type")
	req.Status = c.QueryParam("status")
	req.Search = c.QueryParam("search")

	// Parse integer params with defaults
	page := c.QueryParam("page")
	if page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			req.Page = p
		}
	}
	if req.Page == 0 {
		req.Page = 1
	}

	pageSize := c.QueryParam("page_size")
	if pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil {
			req.PageSize = ps
		}
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	req.SortBy = c.QueryParam("sort_by")
	req.SortOrder = c.QueryParam("sort_order")

	// Parse tags array
	if tags := c.QueryParams()["tags"]; len(tags) > 0 {
		req.Tags = tags
	}

	// Debug logging
	h.logger.Info("ListRooms request",
		zap.String("type", req.Type),
		zap.String("status", req.Status),
		zap.String("search", req.Search),
		zap.Int("page", req.Page),
		zap.Int("page_size", req.PageSize),
	)

	// Build filters
	filters := buildFilters(&req)

	h.logger.Info("ListRooms filters",
		zap.Any("type_filter", filters.Type),
		zap.Any("status_filter", filters.Status),
	)

	rooms, total, err := h.roomService.ListRooms(c.Request().Context(), filters)
	if err != nil {
		return h.handleError(c, errors.ErrInternal(err))
	}

	return h.handleSuccess(c, presenter.ToRoomListResponse(rooms, total, req.Page, req.PageSize))
}

// JoinRoom handles POST /rooms/:id/join
// @Summary      Join a room
// @Description  Allows a user to join an existing room and get LiveKit credentials
// @Tags         Rooms
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Room ID (UUID)"
// @Success      200  {object}  room.JoinRoomResponse  "Successfully joined room with LiveKit credentials"
// @Failure      400  {object}  map[string]interface{}  "Invalid room ID or room is full"
// @Failure      401  {object}  map[string]interface{}  "User not authenticated"
// @Failure      409  {object}  map[string]interface{}  "User already in room"
// @Failure      500  {object}  map[string]interface{}  "Failed to join room"
// @Router       /rooms/{id}/participants [post]
func (h *Room) JoinRoom(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return h.handleError(c, errors.ErrInvalidArgument("Invalid room ID").WithDetail("error", "Room ID must be a valid UUID"))
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return h.handleError(c, errors.ErrUnauthenticated().WithDetail("error", "User not authenticated"))
	}

	input := roomUsecase.JoinRoomInput{
		RoomID: roomID,
		UserID: userID,
	}

	r, participant, err := h.roomService.JoinRoom(c.Request().Context(), input)
	if err != nil {
		return h.handleError(c, errors.ErrInternal(err))
	}

	// Check if user is in waiting room
	if participant.Status == entities.ParticipantStatusWaiting {
		// Return waiting room response (no LiveKit token)
		response := &room.JoinRoomResponse{
			Status:      "waiting",
			Message:     "You are in the waiting room. Waiting for host approval.",
			Room:        presenter.ToRoomResponse(r),
			Participant: presenter.ToParticipantResponse(participant),
		}
		return h.handleSuccess(c, response)
	}

	// User has joined successfully - generate LiveKit token
	var livekitToken string
	livekitToken, err = h.roomService.GenerateParticipantToken(c.Request().Context(), r, participant)
	if err != nil {
		return h.handleError(c, errors.ErrInternal(err))
	}

	response := &room.JoinRoomResponse{
		Status:       "joined",
		Message:      "Successfully joined the room",
		Room:         presenter.ToRoomResponse(r),
		Participant:  presenter.ToParticipantResponse(participant),
		LivekitToken: livekitToken,
		LivekitURL:   h.roomService.GetLivekitURL(),
	}

	return h.handleSuccess(c, response)
}

// LeaveRoom handles POST /rooms/:id/leave
// @Summary      Leave a room
// @Description  Allows a user to leave a room they are currently in
// @Tags         Rooms
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Room ID (UUID)"
// @Success      200  {object}  map[string]interface{}  "Successfully left the room"
// @Failure      400  {object}  map[string]interface{}  "Invalid room ID"
// @Failure      401  {object}  map[string]interface{}  "User not authenticated"
// @Failure      500  {object}  map[string]interface{}  "Failed to leave room"
// @Router       /rooms/{id}/participants/me [delete]
func (h *Room) LeaveRoom(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return h.handleError(c, errors.ErrInvalidArgument("Invalid room ID").WithDetail("error", "Room ID must be a valid UUID"))
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return h.handleError(c, errors.ErrUnauthenticated().WithDetail("error", "User not authenticated"))
	}

	if err := h.roomService.LeaveRoom(c.Request().Context(), roomID, userID); err != nil {
		return h.handleError(c, errors.ErrInternal(err))
	}

	return h.handleSuccess(c, map[string]interface{}{
		"message": "successfully left the room",
	})
}

// EndRoom handles POST /rooms/:id/end
// @Summary      End a room
// @Description  Ends a room session (host only)
// @Tags         Rooms
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Room ID (UUID)"
// @Success      200  {object}  map[string]interface{}  "Room ended successfully"
// @Failure      400  {object}  map[string]interface{}  "Invalid room ID"
// @Failure      401  {object}  map[string]interface{}  "User not authenticated"
// @Failure      403  {object}  map[string]interface{}  "User is not the host"
// @Failure      500  {object}  map[string]interface{}  "Failed to end room"
// @Router       /rooms/{id} [patch]
func (h *Room) EndRoom(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return h.handleError(c, errors.ErrInvalidArgument("Invalid room ID").WithDetail("error", "Room ID must be a valid UUID"))
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return h.handleError(c, errors.ErrUnauthenticated().WithDetail("error", "User not authenticated"))
	}

	if err := h.roomService.EndRoom(c.Request().Context(), roomID, userID); err != nil {
		return h.handleError(c, err)
	}

	return h.handleSuccess(c, map[string]string{"message": "room ended successfully"})
}

// GetParticipants handles GET /rooms/:id/participants
// @Summary      Get room participants
// @Description  Gets a list of all participants in a room
// @Tags         Participants
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Room ID (UUID)"
// @Success      200  {object}  room.ParticipantListResponse  "List of participants"
// @Failure      400  {object}  map[string]interface{}  "Invalid room ID"
// @Failure      500  {object}  map[string]interface{}  "Failed to get participants"
// @Router       /rooms/{id}/participants [get]
func (h *Room) GetParticipants(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return h.handleError(c, errors.ErrInvalidArgument("Invalid room ID").WithDetail("error", "Room ID must be a valid UUID"))
	}

	participants, err := h.roomService.GetParticipants(c.Request().Context(), roomID)
	if err != nil {
		return h.handleError(c, errors.ErrInternal(err))
	}

	return h.handleSuccess(c, presenter.ToParticipantListResponse(participants))
}

// GetWaitingParticipants handles GET /rooms/:id/participants/waiting
// @Summary      Get waiting participants
// @Description  Gets a list of participants waiting for approval in a room (host only)
// @Tags         Participants
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Room ID (UUID)"
// @Success      200  {object}  room.ParticipantListResponse  "List of waiting participants"
// @Failure      400  {object}  map[string]interface{}  "Invalid room ID"
// @Failure      401  {object}  map[string]interface{}  "User not authenticated"
// @Failure      403  {object}  map[string]interface{}  "User is not the host"
// @Failure      500  {object}  map[string]interface{}  "Failed to get waiting participants"
// @Router       /rooms/{id}/participants/waiting [get]
func (h *Room) GetWaitingParticipants(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return h.handleError(c, errors.ErrInvalidArgument("Invalid room ID").WithDetail("error", "Room ID must be a valid UUID"))
	}

	// participantID, err := uuid.Parse(c.Param("pid"))
	// if err != nil {
	// 	return h.handleError(c, errors.ErrInvalidArgument("Invalid room ID").WithDetail("error", "Room ID must be a valid UUID"))
	// }

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return h.handleError(c, errors.ErrUnauthenticated().WithDetail("error", "User not authenticated"))
	}

	participants, err := h.roomService.GetWaitingParticipants(c.Request().Context(), roomID, userID)
	if err != nil {
		return h.handleError(c, err)
	}

	return h.handleSuccess(c, presenter.ToParticipantListResponse(participants))
}

// TransferHost handles POST /rooms/:id/transfer-host
// @Summary      Transfer host role
// @Description  Transfers the host role to another participant (host only)
// @Tags         Participants
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string  true  "Room ID (UUID)"
// @Param        request  body      room.TransferHostRequest  true  "New host user ID"
// @Success      200      {object}  map[string]interface{}  "Host transferred successfully"
// @Failure      400      {object}  map[string]interface{}  "Invalid room ID, new host ID, or new host is not a participant"
// @Failure      401      {object}  map[string]interface{}  "User not authenticated"
// @Failure      403      {object}  map[string]interface{}  "User is not the host"
// @Failure      500      {object}  map[string]interface{}  "Failed to transfer host"
// @Router       /rooms/{id}/host [patch]
func (h *Room) TransferHost(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return h.handleError(c, errors.ErrInvalidArgument("Invalid room ID").WithDetail("error", "Room ID must be a valid UUID"))
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return h.handleError(c, errors.ErrUnauthenticated().WithDetail("error", "User not authenticated"))
	}

	var req room.TransferHostRequest
	if err := c.Bind(&req); err != nil {
		return h.handleError(c, errors.ErrInvalidArgument("Invalid request body").WithDetail("error", err.Error()))
	}

	newHostID, err := uuid.Parse(req.NewHostID)
	if err != nil {
		return h.handleError(c, errors.ErrInvalidArgument("Invalid new host ID").WithDetail("error", "New host ID must be a valid UUID"))
	}

	if err := h.roomService.TransferHost(c.Request().Context(), roomID, userID, newHostID); err != nil {
		return h.handleError(c, err)
	}
	return h.handleSuccess(c, map[string]string{"message": "host transferred successfully"})
}

// AdmitParticipant handles POST /rooms/:id/participants/:pid/admit
// @Summary      Admit participant
// @Description  Admits a waiting participant to join the room (host only)
// @Tags         Participants
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Room ID (UUID)"
// @Param        pid  path      string  true  "Participant ID (UUID)"
// @Success      200  {object}  map[string]interface{}  "Participant admitted successfully"
// @Failure      400  {object}  map[string]interface{}  "Invalid room or participant ID"
// @Failure      401  {object}  map[string]interface{}  "User not authenticated"
// @Failure      403  {object}  map[string]interface{}  "User is not the host"
// @Failure      500  {object}  map[string]interface{}  "Failed to admit participant"
// @Router       /rooms/{id}/participants/{pid}/admit [post]
func (h *Room) AdmitParticipant(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return h.handleError(c, errors.ErrInvalidArgument("Invalid room ID").WithDetail("error", "Room ID must be a valid UUID"))
	}

	participantID, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		return h.handleError(c, errors.ErrInvalidArgument("Invalid participant ID").WithDetail("error", "Participant ID must be a valid UUID"))
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return h.handleError(c, errors.ErrUnauthenticated().WithDetail("error", "User not authenticated"))
	}

	accessToken, err := h.roomService.AdmitParticipant(c.Request().Context(), roomID, userID, participantID)
	if err != nil {
		return h.handleError(c, err)
	}

	return h.handleSuccess(c, map[string]interface{}{
		"message":     "participant admitted successfully",
		"accessToken": accessToken,
	})
}

// DenyParticipant handles POST /rooms/:id/participants/:pid/deny
// @Summary      Deny participant
// @Description  Denies a waiting participant from joining the room (host only)
// @Tags         Rooms
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string  true  "Room ID (UUID)"
// @Param        pid  path      string  true  "Participant ID (UUID)"
// @Success      200  {object}  map[string]interface{}  "Participant denied successfully"
// @Failure      400  {object}  map[string]interface{}  "Invalid room or participant ID"
// @Failure      401  {object}  map[string]interface{}  "User not authenticated"
// @Failure      403  {object}  map[string]interface{}  "User is not the host"
// @Failure      500  {object}  map[string]interface{}  "Failed to deny participant"
// @Router       /rooms/{id}/participants/{pid}/deny [post]
func (h *Room) DenyParticipant(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return h.handleError(c, errors.ErrInvalidArgument("Invalid room ID").WithDetail("error", "Room ID must be a valid UUID"))
	}

	participantID, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		return h.handleError(c, errors.ErrInvalidArgument("Invalid participant ID").WithDetail("error", "Participant ID must be a valid UUID"))
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return h.handleError(c, errors.ErrUnauthenticated().WithDetail("error", "User not authenticated"))
	}

	var req room.DenyParticipantRequest
	if err := c.Bind(&req); err != nil {
		// Reason is optional, so we ignore bind errors
		req.Reason = ""
	}

	if err := h.roomService.DenyParticipant(c.Request().Context(), roomID, userID, participantID, req.Reason); err != nil {
		return h.handleError(c, err)
	}

	return h.handleSuccess(c, map[string]interface{}{
		"message": "participant denied successfully",
	})
}

// RemoveParticipant handles DELETE /rooms/:id/participants/:pid
// @Summary      Remove a participant
// @Description  Removes a participant from the room (host/co-host only)
// @Tags         Participants
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path      string  true  "Room ID (UUID)"
// @Param        pid     path      string  true  "Participant ID (UUID)"
// @Param        request body      room.RemoveParticipantRequest  false  "Reason for removal"
// @Success      200     {object}  map[string]interface{}  "Participant removed successfully"
// @Failure      400     {object}  map[string]interface{}  "Invalid room or participant ID"
// @Failure      401     {object}  map[string]interface{}  "User not authenticated"
// @Failure      403     {object}  map[string]interface{}  "User is not the host"
// @Failure      500     {object}  map[string]interface{}  "Failed to remove participant"
// @Router       /rooms/{id}/participants/{pid} [delete]
func (h *Room) RemoveParticipant(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return h.handleError(c, errors.ErrInvalidArgument("Invalid room ID").WithDetail("error", "Room ID must be a valid UUID"))
	}

	participantID, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		return h.handleError(c, errors.ErrInvalidArgument("Invalid participant ID").WithDetail("error", "Participant ID must be a valid UUID"))
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return h.handleError(c, errors.ErrUnauthenticated().WithDetail("error", "User not authenticated"))
	}

	var req room.RemoveParticipantRequest
	if err := c.Bind(&req); err != nil {
		// Reason is optional; bind errors treated as empty reason
		req.Reason = ""
	}

	if err := h.roomService.RemoveParticipant(c.Request().Context(), roomID, userID, participantID, req.Reason); err != nil {
		return h.handleError(c, err)
	}

	return h.handleSuccess(c, map[string]interface{}{"message": "participant removed successfully"})
}

// InviteByEmail invites a user to join a room by email
// @Summary      Invite user by email
// @Description  Allows the host to invite a user to join the room by providing their email
// @Tags         Invitations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                      true  "Room ID (UUID)"
// @Param        request  body      room.InviteByEmailRequest   true  "Email of user to invite"
// @Success      200      {object}  room.InviteByEmailResponse  "Invitation sent successfully"
// @Failure      400      {object}  map[string]interface{}      "Invalid request"
// @Failure      401      {object}  map[string]interface{}      "User not authenticated"
// @Failure      403      {object}  map[string]interface{}      "User is not the host"
// @Failure      409      {object}  map[string]interface{}      "User already invited"
// @Failure      500      {object}  map[string]interface{}      "Failed to send invitation"
// @Router       /rooms/{id}/invitations [post]
func (h *Room) InviteByEmail(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return h.handleError(c, errors.ErrInvalidArgument("Invalid room ID").WithDetail("error", "Room ID must be a valid UUID"))
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return h.handleError(c, errors.ErrUnauthenticated().WithDetail("error", "User not authenticated"))
	}

	var req room.InviteByEmailRequest
	if err := c.Bind(&req); err != nil {
		return h.handleError(c, errors.ErrInvalidArgument("Invalid request body").WithDetail("error", err.Error()))
	}

	if err := c.Validate(&req); err != nil {
		return h.handleError(c, errors.ErrInvalidArgument("Validation failed").WithDetail("error", err.Error()))
	}

	participant, err := h.roomService.InviteUserByEmail(c.Request().Context(), roomID, userID, req.Email)
	if err != nil {
		return h.handleError(c, err)
	}

	return h.handleSuccess(c, &room.InviteByEmailResponse{
		ParticipantID: participant.ID.String(),
		Email:         req.Email,
		Status:        string(participant.Status),
		Message:       "Invitation sent successfully. The user will see this invitation when they log in.",
	})
}

// GetMyInvitations retrieves all room invitations for the current user
// @Summary      Get my invitations
// @Description  Retrieves all room invitations sent to the current user's email
// @Tags         Invitations
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  room.MyInvitationsResponse  "List of invitations"
// @Failure      401  {object}  map[string]interface{}      "User not authenticated"
// @Failure      500  {object}  map[string]interface{}      "Failed to get invitations"
// @Router       /invitations/me [get]
func (h *Room) GetMyInvitations(c echo.Context) error {
	// Get user email from context (set by auth middleware)
	userEmail, ok := c.Get("user_email").(string)
	if !ok || userEmail == "" {
		return h.handleError(c, errors.ErrUnauthenticated().WithDetail("error", "User email not found"))
	}

	invitations, err := h.roomService.GetInvitationsByEmail(c.Request().Context(), userEmail)
	if err != nil {
		return h.handleError(c, errors.ErrInternal(err))
	}

	return h.handleSuccess(c, presenter.ToMyInvitationsResponse(invitations))
}

// AcceptInvitationToRoom accepts an invitation and joins the room
// @Summary      Accept invitation
// @Description  Accepts an invitation to join a room
// @Tags         Invitations
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                           true  "Room ID (UUID)"
// @Success      200  {object}  room.AcceptInvitationResponse    "Successfully joined room"
// @Failure      400  {object}  map[string]interface{}           "Invalid room ID"
// @Failure      401  {object}  map[string]interface{}           "User not authenticated"
// @Failure      404  {object}  map[string]interface{}           "Invitation not found"
// @Failure      500  {object}  map[string]interface{}           "Failed to accept invitation"
// @Router       /rooms/{id}/invitations/accept [post]
func (h *Room) AcceptInvitationToRoom(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return h.handleError(c, errors.ErrInvalidArgument("Invalid room ID").WithDetail("error", "Room ID must be a valid UUID"))
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return h.handleError(c, errors.ErrUnauthenticated().WithDetail("error", "User not authenticated"))
	}

	userEmail, ok := c.Get("user_email").(string)
	if !ok || userEmail == "" {
		return h.handleError(c, errors.ErrUnauthenticated().WithDetail("error", "User email not found"))
	}

	r, participant, token, err := h.roomService.AcceptInvitationByEmail(c.Request().Context(), roomID, userEmail, userID)
	if err != nil {
		return h.handleError(c, err)
	}

	return h.handleSuccess(c, &room.AcceptInvitationResponse{
		Status:       "joined",
		Message:      "Successfully accepted invitation and joined the room",
		Room:         presenter.ToRoomResponse(r),
		Participant:  presenter.ToParticipantResponse(participant),
		LivekitToken: token,
		LivekitURL:   h.roomService.GetLivekitURL(),
	})
}

// DeclineInvitation declines an invitation to join a room
// @Summary      Decline invitation
// @Description  Declines an invitation to join a room
// @Tags         Invitations
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                      true  "Room ID (UUID)"
// @Success      200  {object}  map[string]interface{}      "Invitation declined"
// @Failure      400  {object}  map[string]interface{}      "Invalid room ID"
// @Failure      401  {object}  map[string]interface{}      "User not authenticated"
// @Failure      404  {object}  map[string]interface{}      "Invitation not found"
// @Failure      500  {object}  map[string]interface{}      "Failed to decline invitation"
// @Router       /rooms/{id}/invitations/decline [post]
func (h *Room) DeclineInvitation(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return h.handleError(c, errors.ErrInvalidArgument("Invalid room ID").WithDetail("error", "Room ID must be a valid UUID"))
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return h.handleError(c, errors.ErrUnauthenticated().WithDetail("error", "User not authenticated"))
	}

	userEmail, ok := c.Get("user_email").(string)
	if !ok || userEmail == "" {
		return h.handleError(c, errors.ErrUnauthenticated().WithDetail("error", "User email not found"))
	}

	if err := h.roomService.DeclineInvitationByEmail(c.Request().Context(), roomID, userEmail, userID); err != nil {
		return h.handleError(c, err)
	}

	return h.handleSuccess(c, map[string]interface{}{
		"message": "Invitation declined successfully",
	})
}

// GetRoomInvitations retrieves all invitations for a room (host only)
// @Summary      Get room invitations
// @Description  Retrieves all invitations for a room (host only)
// @Tags         Invitations
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                         true  "Room ID (UUID)"
// @Success      200  {object}  room.ParticipantListResponse   "List of invitations"
// @Failure      400  {object}  map[string]interface{}         "Invalid room ID"
// @Failure      401  {object}  map[string]interface{}         "User not authenticated"
// @Failure      403  {object}  map[string]interface{}         "User is not the host"
// @Failure      500  {object}  map[string]interface{}         "Failed to get invitations"
// @Router       /rooms/{id}/invitations [get]
func (h *Room) GetRoomInvitations(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return h.handleError(c, errors.ErrInvalidArgument("Invalid room ID").WithDetail("error", "Room ID must be a valid UUID"))
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return h.handleError(c, errors.ErrUnauthenticated().WithDetail("error", "User not authenticated"))
	}

	invitations, err := h.roomService.GetRoomInvitations(c.Request().Context(), roomID, userID)
	if err != nil {
		return h.handleError(c, err)
	}

	return h.handleSuccess(c, presenter.ToParticipantListResponse(invitations))
}

// GetMeetingSummary retrieves the AI-generated summary for a meeting
// @Summary      Get meeting summary
// @Description  Retrieves the AI-generated meeting summary, including key points, decisions, action items, and sentiment analysis
// @Tags         Meetings
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      string                                    true  "Room ID (UUID)"
// @Success      200  {object}  summaryDTO.MeetingSummaryResponse         "Meeting summary retrieved successfully"
// @Success      202  {object}  summaryDTO.SummaryStatusResponse          "Summary is being generated"
// @Failure      400  {object}  map[string]interface{}                    "Invalid room ID"
// @Failure      401  {object}  map[string]interface{}                    "User not authenticated"
// @Failure      404  {object}  map[string]interface{}                    "Meeting not found or no transcript available"
// @Failure      500  {object}  map[string]interface{}                    "Failed to retrieve summary"
// @Router       /meetings/{id}/summary [get]
func (h *Room) GetMeetingSummary(c echo.Context) error {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return h.handleError(c, errors.ErrInvalidArgument("Invalid room ID").WithDetail("error", "Room ID must be a valid UUID"))
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return h.handleError(c, errors.ErrUnauthenticated().WithDetail("error", "User not authenticated"))
	}

	h.logger.Info("Retrieving meeting summary",
		zap.String("room_id", roomID.String()),
		zap.String("user_id", userID.String()),
	)

	// Try to get existing summary
	summary, err := h.summaryRepo.GetMeetingSummaryByRoom(c.Request().Context(), roomID)
	if err == nil && summary != nil {
		// Summary found, prepare response
		response, err := h.buildSummaryResponse(c.Request().Context(), summary)
		if err != nil {
			h.logger.Error("Failed to build summary response",
				zap.String("room_id", roomID.String()),
				zap.Error(err),
			)
			return h.handleError(c, errors.ErrInternal(err).WithDetail("error", "Failed to build summary response"))
		}

		h.logger.Info("Meeting summary retrieved successfully",
			zap.String("room_id", roomID.String()),
		)
		return h.handleSuccess(c, response)
	}

	// No summary yet, check AI job status
	jobs, err := h.aiJobRepo.ListAIJobsByMeetingID(c.Request().Context(), roomID)
	if err != nil {
		h.logger.Error("Failed to retrieve AI job",
			zap.String("room_id", roomID.String()),
			zap.Error(err),
		)
		return h.handleError(c, errors.ErrNotFound("Meeting not found or no transcript available").WithDetail("error", "No AI job found for this meeting"))
	}

	// Find the latest job
	if len(jobs) == 0 {
		return h.handleError(c, errors.ErrNotFound("Meeting not found or no transcript available").WithDetail("error", "No AI job found for this meeting"))
	}

	job := jobs[0] // Latest job (ordered by created_at DESC)

	// Check job status
	switch job.Status {
	case entities.AIJobStatusTranscriptReady, entities.AIJobStatusSummarizing:
		// Summary is being processed
		statusResponse := summaryDTO.SummaryStatusResponse{
			Status:      string(job.Status),
			Message:     "Meeting summary is being generated. Please check back in a few moments.",
			RoomID:      roomID,
			JobID:       job.ID,
			SubmittedAt: job.CreatedAt,
		}

		h.logger.Info("Summary generation in progress",
			zap.String("room_id", roomID.String()),
			zap.String("status", string(job.Status)),
		)

		return c.JSON(202, map[string]interface{}{
			"data": statusResponse,
		})

	case entities.AIJobStatusFailed:
		// Job failed
		failureReason := "Summary generation failed"
		if job.LastError != nil {
			failureReason = *job.LastError
		}

		h.logger.Error("AI job failed",
			zap.String("room_id", roomID.String()),
			zap.String("failure_reason", failureReason),
		)
		return h.handleError(c, errors.ErrInternal(fmt.Errorf("%s", failureReason)).WithDetail("error", failureReason))

	case entities.AIJobStatusCompleted:
		// Job completed but no summary found (shouldn't happen, but handle gracefully)
		h.logger.Warn("AI job completed but no summary found",
			zap.String("room_id", roomID.String()),
		)
		return h.handleError(c, errors.ErrNotFound("Meeting summary not found").WithDetail("error", "Job completed but summary not available"))

	default:
		// Other statuses (pending, submitted)
		h.logger.Info("AI job not ready",
			zap.String("room_id", roomID.String()),
			zap.String("status", string(job.Status)),
		)
		return h.handleError(c, errors.ErrNotFound("Meeting transcript not ready yet").WithDetail("status", string(job.Status)))
	}
}

// buildSummaryResponse converts entities to DTO response
func (h *Room) buildSummaryResponse(ctx context.Context, summary *entities.MeetingSummary) (*summaryDTO.MeetingSummaryResponse, error) {
	response := &summaryDTO.MeetingSummaryResponse{
		ID:               summary.ID,
		RoomID:           summary.RoomID,
		TranscriptID:     summary.TranscriptID,
		ExecutiveSummary: summary.ExecutiveSummary,
		CreatedAt:        summary.CreatedAt,
		UpdatedAt:        summary.UpdatedAt,
	}

	// Parse key points
	if summary.KeyPoints != nil {
		var keyPoints []summaryDTO.KeyPoint
		if err := json.Unmarshal(summary.KeyPoints, &keyPoints); err != nil {
			h.logger.Warn("Failed to parse key points", zap.Error(err))
		} else {
			response.KeyPoints = keyPoints
		}
	}

	// Parse decisions
	if summary.Decisions != nil {
		var decisions []summaryDTO.Decision
		if err := json.Unmarshal(summary.Decisions, &decisions); err != nil {
			h.logger.Warn("Failed to parse decisions", zap.Error(err))
		} else {
			response.Decisions = decisions
		}
	}

	// Parse topics
	if summary.Topics != nil {
		var topics []string
		if err := json.Unmarshal(summary.Topics, &topics); err != nil {
			h.logger.Warn("Failed to parse topics", zap.Error(err))
		} else {
			response.Topics = topics
		}
	}

	// Parse sentiment breakdown
	if summary.SentimentBreakdown != nil {
		var sentimentBreakdown map[string]interface{}
		if err := json.Unmarshal(summary.SentimentBreakdown, &sentimentBreakdown); err != nil {
			h.logger.Warn("Failed to parse sentiment breakdown", zap.Error(err))
		} else {
			response.SentimentBreakdown = sentimentBreakdown
		}
	}

	// Set engagement metrics
	response.EngagementMetrics = summaryDTO.EngagementMetricsDTO{
		TotalSpeakingTime:       summary.TotalSpeakingTime,
		EngagementScore:         summary.EngagementScore,
		ParticipantBalanceScore: summary.ParticipantBalance,
	}

	// Get action items
	actionItems, err := h.summaryRepo.GetActionItemsBySummary(ctx, summary.ID)
	if err != nil {
		h.logger.Warn("Failed to retrieve action items", zap.Error(err))
	} else {
		response.ActionItems = make([]summaryDTO.ActionItemDTO, len(actionItems))
		for i, item := range actionItems {
			var assignedTo *string
			if item.AssignedTo != nil {
				assignedToStr := item.AssignedTo.String()
				assignedTo = &assignedToStr
			}

			response.ActionItems[i] = summaryDTO.ActionItemDTO{
				ID:          item.ID,
				Title:       item.Title,
				Description: item.Description,
				Type:        string(item.Type),
				Priority:    string(item.Priority),
				AssignedTo:  assignedTo,
				DueDate:     item.DueDate,
				Status:      string(item.Status),
				CreatedAt:   item.CreatedAt,
			}
		}
	}

	return response, nil
}
