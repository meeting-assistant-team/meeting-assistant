package handler

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/johnquangdev/meeting-assistant/internal/adapter/dto/room"
	"github.com/johnquangdev/meeting-assistant/internal/adapter/presenter"
	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
	"github.com/johnquangdev/meeting-assistant/internal/domain/repositories"
	usecaseErrors "github.com/johnquangdev/meeting-assistant/internal/usecase/errors"
	roomUsecase "github.com/johnquangdev/meeting-assistant/internal/usecase/room"
)

// Room handles room-related HTTP requests
type Room struct {
	roomService roomUsecase.Service
}

// NewRoomHandler creates a new room handler
func NewRoomHandler(roomService roomUsecase.Service) *Room {
	return &Room{
		roomService: roomService,
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
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "invalid_request",
			"message": err.Error(),
		})
	}

	// Validate request
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "validation_failed",
			"message": err.Error(),
		})
	}

	// Get user ID from context (set by auth middleware)
	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error":   "unauthorized",
			"message": "user not authenticated",
		})
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
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "invalid_room_type",
			"message": "room type must be public, private, or scheduled",
		})
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
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error":   "failed_to_create_room",
			"message": err.Error(),
		})
	}

	response := &room.CreateRoomResponse{
		Room:         presenter.ToRoomResponse(output.Room),
		LivekitToken: output.LivekitToken,
		LivekitURL:   output.LivekitURL,
	}

	return c.JSON(http.StatusCreated, response)
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
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "invalid_room_id",
			"message": "room ID must be a valid UUID",
		})
	}

	r, err := h.roomService.GetRoom(c.Request().Context(), roomID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"error":   "room_not_found",
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, presenter.ToRoomResponse(r))
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
	var req room.ListRoomsRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "invalid_request",
			"message": err.Error(),
		})
	}

	// Set defaults
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	// Build filters
	filters := h.buildFilters(&req)

	rooms, total, err := h.roomService.ListRooms(c.Request().Context(), filters)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error":   "failed_to_list_rooms",
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, presenter.ToRoomListResponse(rooms, total, req.Page, req.PageSize))
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
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "invalid_room_id",
			"message": "room ID must be a valid UUID",
		})
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error":   "unauthorized",
			"message": "user not authenticated",
		})
	}

	input := roomUsecase.JoinRoomInput{
		RoomID: roomID,
		UserID: userID,
	}

	r, participant, err := h.roomService.JoinRoom(c.Request().Context(), input)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "failed_to_join_room"

		// Map specific errors to HTTP status codes
		switch {
		case errors.Is(err, usecaseErrors.ErrRoomFull):
			statusCode = http.StatusBadRequest
			errorCode = "room_full"
		case errors.Is(err, usecaseErrors.ErrAlreadyInRoom):
			statusCode = http.StatusConflict
			errorCode = "already_in_room"
		case errors.Is(err, usecaseErrors.ErrNotInvited):
			statusCode = http.StatusForbidden
			errorCode = "not_invited"
		case errors.Is(err, usecaseErrors.ErrAccessDenied):
			statusCode = http.StatusForbidden
			errorCode = "access_denied"
		case errors.Is(err, usecaseErrors.ErrTooEarly):
			statusCode = http.StatusBadRequest
			errorCode = "too_early"
		case errors.Is(err, usecaseErrors.ErrRoomEnded):
			statusCode = http.StatusBadRequest
			errorCode = "room_ended"
		case errors.Is(err, usecaseErrors.ErrRoomNotFound):
			statusCode = http.StatusNotFound
			errorCode = "room_not_found"
		case errors.Is(err, usecaseErrors.ErrWaitingForHostApproval):
			statusCode = http.StatusAccepted
			errorCode = "waiting_for_host_approval"
			return c.JSON(statusCode, map[string]interface{}{
				"error":   errorCode,
				"message": "Your request to join the room is pending host approval.",
			})
		}

		return c.JSON(statusCode, map[string]interface{}{
			"error":   errorCode,
			"message": err.Error(),
		})
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
		return c.JSON(http.StatusOK, response)
	}

	// User has joined successfully - generate LiveKit token
	livekitToken, err := h.roomService.GenerateParticipantToken(c.Request().Context(), r, participant)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error":   "failed_to_generate_token",
			"message": err.Error(),
		})
	}

	// Return joined response with LiveKit credentials
	response := &room.JoinRoomResponse{
		Status:       "joined",
		Message:      "Successfully joined the room",
		Room:         presenter.ToRoomResponse(r),
		Participant:  presenter.ToParticipantResponse(participant),
		LivekitToken: livekitToken,
		LivekitURL:   h.roomService.GetLivekitURL(),
	}

	return c.JSON(http.StatusOK, response)
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
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "invalid_room_id",
			"message": "room ID must be a valid UUID",
		})
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error":   "unauthorized",
			"message": "user not authenticated",
		})
	}

	if err := h.roomService.LeaveRoom(c.Request().Context(), roomID, userID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error":   "failed_to_leave_room",
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
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
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "invalid_room_id",
			"message": "room ID must be a valid UUID",
		})
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error":   "unauthorized",
			"message": "user not authenticated",
		})
	}

	if err := h.roomService.EndRoom(c.Request().Context(), roomID, userID); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "user is not the host" {
			statusCode = http.StatusForbidden
		}

		return c.JSON(statusCode, map[string]interface{}{
			"error":   "failed_to_end_room",
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "room ended successfully",
	})
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
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "invalid_room_id",
			"message": "room ID must be a valid UUID",
		})
	}

	participants, err := h.roomService.GetParticipants(c.Request().Context(), roomID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error":   "failed_to_get_participants",
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, presenter.ToParticipantListResponse(participants))
}

// GetWaitingParticipants handles GET /rooms/:id/participants/waiting
// @Summary      Get waiting participants
// @Description  Retrieves all participants waiting for host approval (host only)
// @Tags         Rooms
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
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "invalid_room_id",
			"message": "room ID must be a valid UUID",
		})
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error":   "unauthorized",
			"message": "user not authenticated",
		})
	}

	participants, err := h.roomService.GetWaitingParticipants(c.Request().Context(), roomID, userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, usecaseErrors.ErrNotHost) {
			statusCode = http.StatusForbidden
		} else if errors.Is(err, usecaseErrors.ErrRoomNotFound) {
			statusCode = http.StatusNotFound
		}

		return c.JSON(statusCode, map[string]interface{}{
			"error":   "failed_to_get_waiting_participants",
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, presenter.ToParticipantListResponse(participants))
}

// AdmitParticipant handles POST /rooms/:id/participants/:pid/admit
// @Summary      Admit participant
// @Description  Admits a waiting participant into the room (host only)
// @Tags         Rooms
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
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "invalid_room_id",
			"message": "room ID must be a valid UUID",
		})
	}

	participantID, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "invalid_participant_id",
			"message": "participant ID must be a valid UUID",
		})
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error":   "unauthorized",
			"message": "user not authenticated",
		})
	}

	if err := h.roomService.AdmitParticipant(c.Request().Context(), roomID, userID, participantID); err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, usecaseErrors.ErrNotHost) {
			statusCode = http.StatusForbidden
		} else if errors.Is(err, usecaseErrors.ErrRoomNotFound) || errors.Is(err, usecaseErrors.ErrParticipantNotFound) {
			statusCode = http.StatusNotFound
		} else if errors.Is(err, usecaseErrors.ErrInvalidParticipantStatus) {
			statusCode = http.StatusBadRequest
		}

		return c.JSON(statusCode, map[string]interface{}{
			"error":   "failed_to_admit_participant",
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "participant admitted successfully",
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
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "invalid_room_id",
			"message": "room ID must be a valid UUID",
		})
	}

	participantID, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "invalid_participant_id",
			"message": "participant ID must be a valid UUID",
		})
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error":   "unauthorized",
			"message": "user not authenticated",
		})
	}

	var req room.DenyParticipantRequest
	if err := c.Bind(&req); err != nil {
		// Reason is optional, so we ignore bind errors
		req.Reason = ""
	}

	if err := h.roomService.DenyParticipant(c.Request().Context(), roomID, userID, participantID, req.Reason); err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, usecaseErrors.ErrNotHost) {
			statusCode = http.StatusForbidden
		} else if errors.Is(err, usecaseErrors.ErrRoomNotFound) || errors.Is(err, usecaseErrors.ErrParticipantNotFound) {
			statusCode = http.StatusNotFound
		} else if errors.Is(err, usecaseErrors.ErrInvalidParticipantStatus) {
			statusCode = http.StatusBadRequest
		}

		return c.JSON(statusCode, map[string]interface{}{
			"error":   "failed_to_deny_participant",
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
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
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "invalid_room_id",
			"message": "room ID must be a valid UUID",
		})
	}

	participantID, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "invalid_participant_id",
			"message": "participant ID must be a valid UUID",
		})
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error":   "unauthorized",
			"message": "user not authenticated",
		})
	}

	var req room.RemoveParticipantRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "invalid_request",
			"message": err.Error(),
		})
	}

	if err := h.roomService.RemoveParticipant(c.Request().Context(), roomID, userID, participantID, req.Reason); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "user is not the host" {
			statusCode = http.StatusForbidden
		}

		return c.JSON(statusCode, map[string]interface{}{
			"error":   "failed_to_remove_participant",
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "participant removed successfully",
	})
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
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "invalid_room_id",
			"message": "room ID must be a valid UUID",
		})
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error":   "unauthorized",
			"message": "user not authenticated",
		})
	}

	var req room.TransferHostRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "invalid_request",
			"message": err.Error(),
		})
	}

	newHostID, err := uuid.Parse(req.NewHostID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "invalid_new_host_id",
			"message": "new host ID must be a valid UUID",
		})
	}

	if err := h.roomService.TransferHost(c.Request().Context(), roomID, userID, newHostID); err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "user is not the host" {
			statusCode = http.StatusForbidden
		} else if err.Error() == "user is not a participant" {
			statusCode = http.StatusBadRequest
		}

		return c.JSON(statusCode, map[string]interface{}{
			"error":   "failed_to_transfer_host",
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "host transferred successfully",
	})
}

// buildFilters converts ListRoomsRequest to repository filters
func (h *Room) buildFilters(req *room.ListRoomsRequest) repositories.RoomFilters {
	filters := repositories.RoomFilters{
		Search:    req.Search,
		Tags:      req.Tags,
		Limit:     req.PageSize,
		Offset:    (req.Page - 1) * req.PageSize,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	}

	if req.Type != nil {
		roomType := entities.RoomType(*req.Type)
		filters.Type = &roomType
	}

	if req.Status != nil {
		roomStatus := entities.RoomStatus(*req.Status)
		filters.Status = &roomStatus
	}

	return filters
}
