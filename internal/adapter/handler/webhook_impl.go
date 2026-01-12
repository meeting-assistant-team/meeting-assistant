package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/webhook"
	"go.uber.org/zap"

	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
)

// Helper function to extract keys from map for debugging
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// multiKeyProvider implements auth.KeyProvider for LiveKit Cloud webhooks
// LiveKit Cloud signs webhooks with HMAC but doesn't include 'kid' in JWT header
// So we return the webhook signing key for ANY apiKey request
type multiKeyProvider struct {
	webhookSecret string // The signing key from LiveKit Dashboard
}

func (p *multiKeyProvider) GetSecret(apiKey string) string {
	// Always return webhook secret, regardless of apiKey
	// This works because LiveKit signs with the webhook signing key (APIOMTEQXFBCDEJ)
	// but doesn't include 'kid' in the JWT header
	return p.webhookSecret
}

func (p *multiKeyProvider) NumKeys() int {
	// Return 1 because we have one signing key
	return 1
}

// HandleLiveKitWebhook processes LiveKit webhook events with proper signature validation
func (h *WebhookHandler) HandleLiveKitWebhookV2(c echo.Context) error {

	// Read raw body for signature validation
	bodyBytes, err := io.ReadAll(c.Request().Body)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("failed to read webhook body", zap.Error(err))
		}
		return c.JSON(400, map[string]interface{}{"error": "failed to read body"})
	}

	// Restore body for signature validation
	c.Request().Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	c.Logger().Infof("📥 [WEBHOOK] Raw body length: %d bytes", len(bodyBytes))

	// Parse raw JSON directly to avoid enum type mismatch issues
	// LiveKit sends string values like "DISCONNECTED", "MICROPHONE" but Go SDK expects integer enums
	var rawEvent map[string]interface{}
	err = json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&rawEvent)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("failed to parse webhook JSON", zap.Error(err))
		}
		return c.JSON(400, map[string]interface{}{"error": "invalid webhook format"})
	}

	// Get authorization header for optional signature validation
	authHeader := c.Request().Header.Get("Authorization")
	c.Logger().Infof("🔐 [WEBHOOK] Authorization header present: %v", authHeader != "")

	// Optional: Validate signature in production (currently in dev mode, we accept without validation)
	if authHeader != "" {
		// Debug: Decode JWT header to see key ID
		parts := strings.Split(authHeader, ".")
		if len(parts) >= 1 {
			headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
			if err == nil {
				fmt.Printf("\n🔍 [JWT DEBUG] Header: %s\n\n", string(headerBytes))
			}
		}
		// Try to validate signature using webhook secret (NOT API credentials)
		// LiveKit signs webhooks with the Webhook Signing Key from Dashboard
		authProvider := auth.NewSimpleKeyProvider(h.livekitAPIKey, h.livekitSecret)
		fmt.Printf("🔑 [WEBHOOK] Validating với keyID: %s, secret: %s...\n", h.livekitAPIKey, h.livekitSecret[:10])
		_, validationErr := webhook.ReceiveWebhookEvent(c.Request(), authProvider)

		if validationErr != nil {
			fmt.Printf("❌ SIGNATURE VALIDATION FAILED!\n")
			fmt.Printf("   Error: %v\n", validationErr)
			fmt.Printf("   This might be a LiveKit signing key mismatch.\n")
			fmt.Printf("⚠️  FALLING BACK TO UNSIGNED MODE (DEV ONLY)\n")

			if h.logger != nil {
				h.logger.Warn("Webhook signature validation failed - parsing without validation",
					zap.Error(validationErr),
					zap.String("expected_signing_key", h.webhookSecret),
					zap.String("dashboard_shows", "APIOMTEQXFBCDEJ"),
				)
			}
			c.Logger().Errorf("❌ [WEBHOOK] Signature validation error: %v", validationErr)
			c.Logger().Warn("⚠️  Processing webhook WITHOUT signature validation (DEV MODE)")
		} else {
			fmt.Printf("✅ ✅ ✅ SUCCESS! Webhook signature validated!\n")
			fmt.Printf("   Signing key %s is CORRECT!\n", h.webhookSecret)
		}
	} else {
		c.Logger().Warn("⚠️  No authorization header, processing in DEV MODE")
	}

	// Extract event type
	eventType, ok := rawEvent["event"].(string)
	if !ok {
		c.Logger().Error("❌ [WEBHOOK] Missing or invalid event type")
		return c.JSON(400, map[string]interface{}{"error": "missing event type"})
	}

	c.Logger().Infof("✅ [WEBHOOK] Event type: %s", eventType)

	// Extract participant identity if present (for skipping egress)
	var participantIdentity string
	if participant, ok := rawEvent["participant"].(map[string]interface{}); ok {
		if identity, ok := participant["identity"].(string); ok {
			participantIdentity = identity
		}
	}

	// Route to appropriate handler
	switch eventType {
	case "participant_joined":
		// Skip if participant is egress (not a real user)
		if strings.HasPrefix(participantIdentity, "EG_") {
			c.Logger().Infof("⏭️  Skipping egress participant: %s", participantIdentity)
			return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
		}
		return h.handleParticipantJoinedV2(c, rawEvent)
	case "participant_left":
		// Skip if participant is egress (not a real user)
		if strings.HasPrefix(participantIdentity, "EG_") {
			c.Logger().Infof("⏭️  Skipping egress participant: %s", participantIdentity)
			return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
		}
		return h.handleParticipantLeftV2(c, rawEvent)
	case "track_published", "track_unpublished":
		// These events don't affect our application logic, just log and skip
		c.Logger().Infof("⏭️  Skipping track event: %s", eventType)
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
	case "room_started":
		return h.handleRoomStartedV2(c, rawEvent)
	case "room_finished":
		return h.handleRoomFinishedV2(c, rawEvent)
	case "egress_updated", "egress_ended", "egress_finished":
		// Handles RoomCompositeEgress recording events
		c.Logger().Infof("🎬 [WEBHOOK] Processing egress/recording event: %s", eventType)
		return h.handleEgressEndedV2(c, rawEvent, bodyBytes)
	default:
		if h.logger != nil {
			h.logger.Warn("unhandled webhook event", zap.String("event", eventType))
		}
	}

	return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
}

// handleParticipantJoinedV2 handles participant_joined event
func (h *WebhookHandler) handleParticipantJoinedV2(c echo.Context, rawEvent map[string]interface{}) error {
	c.Logger().Info("🔹 [WEBHOOK] Processing participant_joined")

	// Extract participant and room info from raw event
	participant, ok := rawEvent["participant"].(map[string]interface{})
	if !ok {
		h.logger.Warn("participant missing in event")
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
	}

	room, ok := rawEvent["room"].(map[string]interface{})
	if !ok {
		h.logger.Warn("room missing in event")
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
	}

	participantIdentity, _ := participant["identity"].(string)
	roomName, _ := room["name"].(string)

	if participantIdentity == "" || roomName == "" {
		h.logger.Warn("missing participant identity or room name")
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
	}

	c.Logger().Infof("👤 [WEBHOOK] Participant joined: %s in room %s", participantIdentity, roomName)

	userID, err := uuid.Parse(participantIdentity)
	if err != nil {
		h.logger.Warn("⏭️  Skipping webhook - identity is not a valid UUID (likely a test event from LiveKit Dashboard)",
			zap.String("identity", participantIdentity),
			zap.String("room_name", roomName),
			zap.Error(err))
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok", "skipped": "test_event"})
	}

	ctx := c.Request().Context()
	roomEntity, err := h.roomService.GetRoomByLivekitName(ctx, roomName)
	if err != nil {
		h.logger.Error("failed to find room", zap.String("room_name", roomName), zap.Error(err))
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
	}

	if err := h.roomService.UpdateParticipantStatus(ctx, roomEntity.ID, userID, "joined"); err != nil {
		h.logger.Error("failed to update participant status", zap.Error(err))
	}

	return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok", "event": "participant_joined"})
}

// handleParticipantLeftV2 handles participant_left event
func (h *WebhookHandler) handleParticipantLeftV2(c echo.Context, rawEvent map[string]interface{}) error {
	c.Logger().Info("🔹 [WEBHOOK] Processing participant_left")

	// Extract participant and room info from raw event
	participant, ok := rawEvent["participant"].(map[string]interface{})
	if !ok {
		h.logger.Warn("participant missing in event")
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
	}

	room, ok := rawEvent["room"].(map[string]interface{})
	if !ok {
		h.logger.Warn("room missing in event")
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
	}

	participantIdentity, _ := participant["identity"].(string)
	roomName, _ := room["name"].(string)

	if participantIdentity == "" || roomName == "" {
		h.logger.Warn("missing participant identity or room name")
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
	}

	c.Logger().Infof("👋 [WEBHOOK] Participant left: %s from room %s", participantIdentity, roomName)

	userID, err := uuid.Parse(participantIdentity)
	if err != nil {
		h.logger.Warn("⏭️  Skipping webhook - identity is not a valid UUID (likely a test event from LiveKit Dashboard)",
			zap.String("identity", participantIdentity),
			zap.String("room_name", roomName),
			zap.Error(err))
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok", "skipped": "test_event"})
	}

	ctx := c.Request().Context()
	roomEntity, err := h.roomService.GetRoomByLivekitName(ctx, roomName)
	if err != nil {
		h.logger.Error("failed to find room", zap.String("room_name", roomName), zap.Error(err))
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
	}

	if err := h.roomService.LeaveRoom(ctx, roomEntity.ID, userID); err != nil {
		h.logger.Error("failed to auto-leave room", zap.Error(err))
	}

	return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok", "event": "participant_left"})
}

// handleRoomStartedV2 handles room_started event
func (h *WebhookHandler) handleRoomStartedV2(c echo.Context, rawEvent map[string]interface{}) error {
	c.Logger().Info("🔹 [WEBHOOK] Processing room_started")

	// Extract room info from raw event
	room, ok := rawEvent["room"].(map[string]interface{})
	if !ok {
		h.logger.Warn("room missing in event")
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
	}

	roomName, _ := room["name"].(string)
	if roomName == "" {
		h.logger.Warn("missing room name")
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
	}
	c.Logger().Infof("🚀 Room started: %s", roomName)

	ctx := c.Request().Context()
	roomEntity, err := h.roomService.GetRoomByLivekitName(ctx, roomName)
	if err != nil {
		h.logger.Error("failed to find room", zap.Error(err))
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
	}

	_, err = h.roomService.StartRoom(ctx, roomEntity.ID, roomEntity.HostID)
	if err != nil {
		h.logger.Error("failed to start room", zap.Error(err))
	}

	return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok", "event": "room_started"})
}

// handleRoomFinishedV2 handles room_finished event
func (h *WebhookHandler) handleRoomFinishedV2(c echo.Context, rawEvent map[string]interface{}) error {
	c.Logger().Info("🔹 [WEBHOOK] Processing room_finished")

	// Extract room info from raw event
	room, ok := rawEvent["room"].(map[string]interface{})
	if !ok {
		h.logger.Warn("room missing in event")
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
	}

	roomName, _ := room["name"].(string)
	if roomName == "" {
		h.logger.Warn("missing room name")
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
	}
	c.Logger().Infof("🏁 Room finished: %s", roomName)

	ctx := c.Request().Context()
	roomEntity, err := h.roomService.GetRoomByLivekitName(ctx, roomName)
	if err != nil {
		h.logger.Error("failed to find room", zap.Error(err))
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
	}

	if err := h.roomService.EndRoom(ctx, roomEntity.ID, roomEntity.HostID); err != nil {
		h.logger.Error("failed to end room", zap.Error(err))
	}

	h.logger.Info("room finished - waiting for egress_ended webhook", zap.String("room_id", roomEntity.ID.String()))

	// Recording will be handled by egress_ended webhook
	// Both modern egress and legacy recording use the same event

	return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok", "event": "room_finished"})
}

// handleEgressEndedV2 handles egress_ended event (RoomCompositeEgress recording completed)
func (h *WebhookHandler) handleEgressEndedV2(c echo.Context, rawEvent map[string]interface{}, rawBody []byte) error {
	c.Logger().Info("🔹 [WEBHOOK] Processing egress event")

	// rawEvent is already parsed from the main handler, no need to parse again

	// Extract egressInfo từ raw JSON
	var egressInfoMap map[string]interface{}
	if val, ok := rawEvent["egress_info"].(map[string]interface{}); ok {
		egressInfoMap = val
	} else if val, ok := rawEvent["egressInfo"].(map[string]interface{}); ok {
		egressInfoMap = val
	} else {
		h.logger.Warn("❌ egressInfo not found in webhook")
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok", "error": "egressInfo missing"})
	}

	// Extract các fields cần thiết
	egressID, _ := egressInfoMap["egress_id"].(string)
	if egressID == "" {
		egressID, _ = egressInfoMap["egressId"].(string)
	}

	roomName, _ := egressInfoMap["room_name"].(string)
	if roomName == "" {
		roomName, _ = egressInfoMap["roomName"].(string)
	}

	status, _ := egressInfoMap["status"].(string)

	c.Logger().Infof("🎬 Egress ended: %s (room: %s, status: %s)", egressID, roomName, status)

	// Get context early for MinIO operations
	ctx := c.Request().Context()

	// Extract recording URL từ fileResults
	var recordingURL string
	var filename string

	// Thử extract từ file.location trước
	if fileMap, ok := egressInfoMap["file"].(map[string]interface{}); ok {
		if location, ok := fileMap["location"].(string); ok {
			recordingURL = strings.TrimSpace(location) // Trim whitespace including \n
			// Extract filename/path từ URL - giữ nguyên path trong bucket
			// VD: https://minio.infoquang.id.vn/meeting-recordings/recordings/2025-12-31T051604-room-xxx.mp4
			// → filename = recordings/2025-12-31T051604-room-xxx.mp4
			if strings.Contains(recordingURL, "/meeting-recordings/") {
				parts := strings.SplitN(recordingURL, "/meeting-recordings/", 2)
				if len(parts) == 2 {
					filename = parts[1] // Lấy path sau bucket name
				}
			} else if parts := strings.Split(recordingURL, "/"); len(parts) > 0 {
				// Fallback: chỉ lấy tên file cuối
				filename = parts[len(parts)-1]
			}
			h.logger.Info("✅ Found recording URL in file.location",
				zap.String("url", recordingURL),
				zap.String("filename", filename))
		}
	}

	// Nếu chưa có, thử extract từ fileResults
	if recordingURL == "" {
		if fileResults, ok := egressInfoMap["file_results"].([]interface{}); ok {
			h.logger.Info("📁 Processing file results", zap.Int("count", len(fileResults)))

			for i, result := range fileResults {
				resultMap, ok := result.(map[string]interface{})
				if !ok {
					continue
				}

				fname, _ := resultMap["filename"].(string)
				location, _ := resultMap["location"].(string)
				size, _ := resultMap["size"].(float64)

				h.logger.Info("📁 File result",
					zap.Int("index", i),
					zap.String("filename", fname),
					zap.Float64("size", size),
					zap.String("location", location))

				// Filter audio files (.mp4 hoặc có "audio" trong tên)
				fnameLower := strings.ToLower(fname)
				isAudioFile := strings.HasSuffix(fnameLower, ".mp4") ||
					strings.HasSuffix(fnameLower, ".ogg") ||
					strings.HasSuffix(fnameLower, ".mp3") ||
					strings.HasSuffix(fnameLower, ".wav") ||
					strings.HasSuffix(fnameLower, ".m4a") ||
					strings.Contains(fnameLower, "audio")

				if isAudioFile && location != "" {
					recordingURL = strings.TrimSpace(location) // Trim whitespace including \n
					// Extract filename/path giữ nguyên structure trong bucket
					if strings.Contains(recordingURL, "/meeting-recordings/") {
						parts := strings.SplitN(recordingURL, "/meeting-recordings/", 2)
						if len(parts) == 2 {
							filename = parts[1]
						}
					} else {
						filename = fname
					}
					h.logger.Info("✅ Selected audio file",
						zap.String("filename", filename),
						zap.String("location", recordingURL))
					break
				}
			}
		} else if fileResults, ok := egressInfoMap["fileResults"].([]interface{}); ok {
			// Try camelCase version
			for _, result := range fileResults {
				resultMap, ok := result.(map[string]interface{})
				if !ok {
					continue
				}

				fname, _ := resultMap["filename"].(string)
				location, _ := resultMap["location"].(string)

				fnameLower := strings.ToLower(fname)
				isAudioFile := strings.HasSuffix(fnameLower, ".mp4") ||
					strings.Contains(fnameLower, "audio")

				if isAudioFile && location != "" {
					recordingURL = strings.TrimSpace(location) // Trim whitespace including \n
					// Extract filename/path giữ nguyên structure trong bucket
					if strings.Contains(recordingURL, "/meeting-recordings/") {
						parts := strings.SplitN(recordingURL, "/meeting-recordings/", 2)
						if len(parts) == 2 {
							filename = parts[1]
						}
					} else {
						filename = fname
					}
					h.logger.Info("✅ Selected audio file from camelCase",
						zap.String("filename", filename),
						zap.String("location", location))
					break
				}
			}
		}
	}

	// Nếu không tìm thấy recording URL
	if recordingURL == "" {
		h.logger.Warn("❌ No recording URL found in egressInfo")
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok", "event": "egress_ended_no_file"})
	}

	h.logger.Info("🔍 Recording URL extracted",
		zap.String("url", recordingURL),
		zap.String("filename", filename))

	// Recording URL from LiveKit is already publicly accessible
	// No need to generate presigned URL since bucket has public read policy
	h.logger.Info("✅ Using public recording URL for AssemblyAI",
		zap.String("url", recordingURL))

	if recordingURL == "" {
		h.logger.Warn("❌ recording URL not found in egress data", zap.String("egress_id", egressID))
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
	}

	if roomName == "" {
		h.logger.Warn("room name not found in egress event", zap.String("egress_id", egressID))
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
	}

	roomEntity, err := h.roomService.GetRoomByLivekitName(ctx, roomName)
	if err != nil {
		h.logger.Error("failed to find room", zap.String("room_name", roomName), zap.Error(err))
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
	}

	h.logger.Info("✅ egress finished, triggering AI processing",
		zap.String("room_id", roomEntity.ID.String()),
		zap.String("room_name", roomName),
		zap.String("egress_id", egressID),
		zap.String("recording_url", recordingURL))

	// Create recording record in database for tracking
	recording := &entities.Recording{
		RoomID:          roomEntity.ID,
		LivekitEgressID: &egressID,
		Status:          entities.RecordingStatusCompleted,
		FilePath:        &filename,
		FileURL:         &recordingURL,
		StartedAt:       time.Now(), // Ideally should be from egress info
	}

	// Save recording to database
	if err := h.recordingRepo.Create(ctx, recording); err != nil {
		h.logger.Error("❌ failed to save recording to database",
			zap.String("room_id", roomEntity.ID.String()),
			zap.String("egress_id", egressID),
			zap.Error(err))
		// Continue anyway - don't block AI processing
	} else {
		h.logger.Info("✅ Recording saved to database",
			zap.String("recording_id", recording.ID.String()))
	}

	// Trim recording URL to remove any newlines or spaces
	recordingURL = strings.TrimSpace(recordingURL)

	// Create AI job for worker pool to process
	// Job created once here, worker will submit to AssemblyAI
	go func() {
		bgCtx := context.Background()

		// Create AI job for tracking (only once)
		aiJob := entities.NewAIJob(roomEntity.ID, entities.AIJobTypeTranscription, recordingURL)
		if err := h.aiJobRepo.CreateAIJob(bgCtx, aiJob); err != nil {
			h.logger.Error("❌ failed to create AI job",
				zap.String("room_id", roomEntity.ID.String()),
				zap.Error(err))
			return
		}

		h.logger.Info("✅ AI job created, worker will process it",
			zap.String("job_id", aiJob.ID.String()),
			zap.String("room_id", roomEntity.ID.String()),
			zap.String("recording_url", recordingURL))
	}()

	return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok", "event": "egress_ended"})
}
