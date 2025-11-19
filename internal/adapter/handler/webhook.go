package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	roomUsecase "github.com/johnquangdev/meeting-assistant/internal/usecase/room"
)

// WebhookHandler handles LiveKit webhook events
type WebhookHandler struct {
	roomService   roomUsecase.Service
	webhookSecret string
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(roomService roomUsecase.Service, webhookSecret string) *WebhookHandler {
	return &WebhookHandler{
		roomService:   roomService,
		webhookSecret: webhookSecret,
	}
}

// HandleLiveKitWebhook processes LiveKit webhook events
// @Summary      LiveKit Webhook
// @Description  Receives webhook events from LiveKit server
// @Tags         Webhooks
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Router       /webhooks/livekit [post]
func (h *WebhookHandler) HandleLiveKitWebhook(c echo.Context) error {
	c.Logger().Info("🌐 [WEBHOOK] === Received webhook request ===")

	// Read raw body for debugging
	bodyBytes, err := io.ReadAll(c.Request().Body)
	if err != nil {
		c.Logger().Errorf("❌ [WEBHOOK] Failed to read body: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "failed_to_read_body",
			"message": err.Error(),
		})
	}

	c.Logger().Infof("📥 [WEBHOOK] Raw body length: %d bytes", len(bodyBytes))
	c.Logger().Infof("📥 [WEBHOOK] Raw body: %s", string(bodyBytes))

	var payload LiveKitWebhookPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		c.Logger().Errorf("❌ [WEBHOOK] Failed to unmarshal webhook payload: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "invalid_payload",
			"message": err.Error(),
		})
	}

	// Log webhook event for debugging
	c.Logger().Infof("📡 [WEBHOOK] Event type: %s", payload.Event)
	c.Logger().Infof("📡 [WEBHOOK] Room: %v", payload.Room)
	c.Logger().Infof("📡 [WEBHOOK] Participant: %v", payload.Participant)

	// Handle different event types
	switch payload.Event {
	case "participant_joined":
		c.Logger().Info("➡️  [WEBHOOK] Routing to handleParticipantJoined")
		return h.handleParticipantJoined(c, &payload)
	case "participant_left":
		c.Logger().Info("➡️  [WEBHOOK] Routing to handleParticipantLeft")
		return h.handleParticipantLeft(c, &payload)
	case "room_started":
		c.Logger().Info("➡️  [WEBHOOK] Routing to handleRoomStarted")
		return h.handleRoomStarted(c, &payload)
	case "room_finished":
		c.Logger().Info("➡️  [WEBHOOK] Routing to handleRoomFinished")
		return h.handleRoomFinished(c, &payload)
	default:
		// Log but don't fail on unknown events
		c.Logger().Warnf("⚠️  [WEBHOOK] Unhandled webhook event: %s", payload.Event)
	}

	c.Logger().Info("✅ [WEBHOOK] === Webhook processed successfully ===")
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "ok",
	})
}

// handleParticipantJoined handles when a participant joins
func (h *WebhookHandler) handleParticipantJoined(c echo.Context, payload *LiveKitWebhookPayload) error {
	c.Logger().Info("🔹 [WEBHOOK] handleParticipantJoined called")

	participantIdentity, ok := payload.Participant["identity"].(string)
	if !ok {
		c.Logger().Warn("⚠️  [WEBHOOK] participant identity not found in payload")
		return c.JSON(http.StatusOK, map[string]interface{}{"status": "ok"})
	}

	roomName, ok := payload.Room["name"].(string)
	if !ok {
		c.Logger().Warn("⚠️  [WEBHOOK] room name not found in payload")
		return c.JSON(http.StatusOK, map[string]interface{}{"status": "ok"})
	}

	c.Logger().Infof("👤 [WEBHOOK] Participant joined: %s in room %s", participantIdentity, roomName)

	// Parse user ID from participant identity
	userID, err := uuid.Parse(participantIdentity)
	if err != nil {
		c.Logger().Errorf("❌ [WEBHOOK] Failed to parse user ID '%s': %v", participantIdentity, err)
		return c.JSON(http.StatusOK, map[string]interface{}{"status": "ok"})
	}

	// Find room by LiveKit room name
	ctx := c.Request().Context()
	c.Logger().Infof("🔍 [WEBHOOK] Looking for room with livekit_room_name: %s", roomName)

	roomEntity, err := h.roomService.GetRoomByLivekitName(ctx, roomName)
	if err != nil {
		c.Logger().Errorf("❌ [WEBHOOK] Failed to find room '%s': %v", roomName, err)
		return c.JSON(http.StatusOK, map[string]interface{}{"status": "ok"})
	}

	c.Logger().Infof("✅ [WEBHOOK] Found room ID: %s, Name: %s", roomEntity.ID, roomEntity.Name)

	// Update participant status to "joined" if not already
	c.Logger().Infof("💾 [WEBHOOK] Updating participant status for user %s in room %s", userID, roomEntity.ID)

	if err := h.roomService.UpdateParticipantStatus(ctx, roomEntity.ID, userID, "joined"); err != nil {
		c.Logger().Errorf("❌ [WEBHOOK] Failed to update participant status: %v", err)
	} else {
		c.Logger().Infof("✅ [WEBHOOK] Successfully updated participant status to 'joined'")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "ok",
		"event":  "participant_joined",
	})
}

// handleParticipantLeft handles when a participant leaves (or disconnects)
func (h *WebhookHandler) handleParticipantLeft(c echo.Context, payload *LiveKitWebhookPayload) error {
	c.Logger().Info("🔹 [WEBHOOK] handleParticipantLeft called")

	participantIdentity, ok := payload.Participant["identity"].(string)
	if !ok {
		c.Logger().Warn("⚠️  [WEBHOOK] participant identity not found in payload")
		return c.JSON(http.StatusOK, map[string]interface{}{"status": "ok"})
	}

	roomName, ok := payload.Room["name"].(string)
	if !ok {
		c.Logger().Warn("⚠️  [WEBHOOK] room name not found in payload")
		return c.JSON(http.StatusOK, map[string]interface{}{"status": "ok"})
	}

	c.Logger().Infof("👋 [WEBHOOK] Participant left: %s from room %s", participantIdentity, roomName)

	// Parse user ID from participant identity
	userID, err := uuid.Parse(participantIdentity)
	if err != nil {
		c.Logger().Errorf("❌ [WEBHOOK] Failed to parse user ID '%s': %v", participantIdentity, err)
		return c.JSON(http.StatusOK, map[string]interface{}{"status": "ok"})
	}

	// Find room by LiveKit room name
	ctx := c.Request().Context()
	c.Logger().Infof("🔍 [WEBHOOK] Looking for room with livekit_room_name: %s", roomName)

	roomEntity, err := h.roomService.GetRoomByLivekitName(ctx, roomName)
	if err != nil {
		c.Logger().Errorf("❌ [WEBHOOK] Failed to find room '%s': %v", roomName, err)
		return c.JSON(http.StatusOK, map[string]interface{}{"status": "ok"})
	}

	c.Logger().Infof("✅ [WEBHOOK] Found room ID: %s, Name: %s", roomEntity.ID, roomEntity.Name)

	// Auto leave room for the participant
	c.Logger().Infof("💾 [WEBHOOK] Auto-leaving user %s from room %s", userID, roomEntity.ID)

	if err := h.roomService.LeaveRoom(ctx, roomEntity.ID, userID); err != nil {
		c.Logger().Errorf("❌ [WEBHOOK] Failed to auto-leave room: %v", err)
		// Don't fail - participant might have already left via API
	} else {
		c.Logger().Infof("✅ [WEBHOOK] Auto-left user %s from room %s", userID, roomEntity.ID)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "ok",
		"event":  "participant_left",
	})
}

// handleRoomStarted handles when a room starts
func (h *WebhookHandler) handleRoomStarted(c echo.Context, payload *LiveKitWebhookPayload) error {
	roomName, _ := payload.Room["name"].(string)
	c.Logger().Infof("🚀 Room started: %s", roomName)

	ctx := c.Request().Context()
	roomEntity, err := h.roomService.GetRoomByLivekitName(ctx, roomName)
	if err != nil {
		c.Logger().Errorf("Failed to find room: %v", err)
		return c.JSON(http.StatusOK, map[string]interface{}{"status": "ok"})
	}

	// Ensure room status is active
	_, err = h.roomService.StartRoom(ctx, roomEntity.ID, roomEntity.HostID)
	if err != nil {
		c.Logger().Errorf("Failed to start room: %v", err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "ok",
		"event":  "room_started",
	})
}

// handleRoomFinished handles when a room ends (all participants left)
func (h *WebhookHandler) handleRoomFinished(c echo.Context, payload *LiveKitWebhookPayload) error {
	roomName, _ := payload.Room["name"].(string)
	c.Logger().Infof("🏁 Room finished: %s", roomName)

	ctx := c.Request().Context()
	roomEntity, err := h.roomService.GetRoomByLivekitName(ctx, roomName)
	if err != nil {
		c.Logger().Errorf("Failed to find room: %v", err)
		return c.JSON(http.StatusOK, map[string]interface{}{"status": "ok"})
	}

	// Auto-end the room
	if err := h.roomService.EndRoom(ctx, roomEntity.ID, roomEntity.HostID); err != nil {
		c.Logger().Errorf("Failed to end room: %v", err)
	} else {
		c.Logger().Infof("✅ Auto-ended room %s", roomEntity.ID)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "ok",
		"event":  "room_finished",
	})
}

// LiveKitWebhookPayload represents the webhook payload structure
type LiveKitWebhookPayload struct {
	Event       string                 `json:"event"`
	Room        map[string]interface{} `json:"room,omitempty"`
	Participant map[string]interface{} `json:"participant,omitempty"`
	Track       map[string]interface{} `json:"track,omitempty"`
	CreatedAt   string                 `json:"createdAt"` // LiveKit sends this as string, not int64
}

// String returns JSON representation of the payload
func (p *LiveKitWebhookPayload) String() string {
	b, _ := json.MarshalIndent(p, "", "  ")
	return string(b)
}
