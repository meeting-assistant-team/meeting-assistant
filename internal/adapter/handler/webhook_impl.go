package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/webhook"
	"go.uber.org/zap"

	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
)

// HandleLiveKitWebhook processes LiveKit webhook events with proper signature validation
func (h *WebhookHandler) HandleLiveKitWebhookV2(c echo.Context) error {
	c.Logger().Info("🌐 [WEBHOOK] === Received webhook request ===")

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

	// Get authorization header
	authHeader := c.Request().Header.Get("Authorization")
	c.Logger().Infof("🔐 [WEBHOOK] Authorization header: %s", authHeader)

	var event *livekit.WebhookEvent

	if authHeader != "" {
		// Validate webhook signature using LiveKit SDK
		// LiveKit webhook uses API Key and Secret for validation
		authProvider := auth.NewSimpleKeyProvider(h.livekitAPIKey, h.livekitSecret)
		event, err = webhook.ReceiveWebhookEvent(c.Request(), authProvider)
		if err != nil {
			if h.logger != nil {
				h.logger.Error("failed to validate webhook signature", zap.Error(err))
			}
			// Fallback to JSON parsing for development/testing
			c.Logger().Warn("⚠️  Signature validation failed, trying JSON parse (DEV MODE)")
			var eventData livekit.WebhookEvent
			err = json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&eventData)
			if err != nil {
				if h.logger != nil {
					h.logger.Error("failed to parse webhook JSON", zap.Error(err))
				}
				return c.JSON(400, map[string]interface{}{"error": "invalid webhook format"})
			}
			event = &eventData
		}
	} else {
		// No auth header - try JSON parsing for development/testing
		c.Logger().Warn("⚠️  No authorization header, trying JSON parse (DEV MODE)")
		var eventData livekit.WebhookEvent
		err = json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&eventData)
		if err != nil {
			if h.logger != nil {
				h.logger.Error("failed to parse webhook JSON", zap.Error(err))
			}
			return c.JSON(400, map[string]interface{}{"error": "invalid webhook format or missing auth header"})
		}
		event = &eventData
	}

	// Log webhook event
	c.Logger().Infof("✅ [WEBHOOK] Event type: %s", event.Event)

	// Route to appropriate handler
	switch event.Event {
	case "participant_joined":
		// Skip if participant is egress (not a real user)
		if event.Participant != nil && strings.HasPrefix(event.Participant.Identity, "EG_") {
			c.Logger().Infof("⏭️  Skipping egress participant: %s", event.Participant.Identity)
			return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
		}
		return h.handleParticipantJoinedV2(c, event)
	case "participant_left":
		// Skip if participant is egress (not a real user)
		if event.Participant != nil && strings.HasPrefix(event.Participant.Identity, "EG_") {
			c.Logger().Infof("⏭️  Skipping egress participant: %s", event.Participant.Identity)
			return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
		}
		return h.handleParticipantLeftV2(c, event)
	case "room_started":
		return h.handleRoomStartedV2(c, event)
	case "room_finished":
		return h.handleRoomFinishedV2(c, event)
	case "egress_ended", "egress_finished":
		c.Logger().Infof("🎬 [WEBHOOK] Processing egress event: %s", event.Event)
		return h.handleEgressEndedV2(c, event)
	default:
		if h.logger != nil {
			h.logger.Warn("unhandled webhook event", zap.String("event", event.Event))
		}
	}

	return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
}

// handleParticipantJoinedV2 handles participant_joined event
func (h *WebhookHandler) handleParticipantJoinedV2(c echo.Context, event *livekit.WebhookEvent) error {
	c.Logger().Info("🔹 [WEBHOOK] Processing participant_joined")

	if event.Participant == nil || event.Room == nil {
		h.logger.Warn("participant or room missing in event")
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
	}

	participantIdentity := event.Participant.Identity
	roomName := event.Room.Name

	c.Logger().Infof("👤 [WEBHOOK] Participant joined: %s in room %s", participantIdentity, roomName)

	userID, err := uuid.Parse(participantIdentity)
	if err != nil {
		h.logger.Error("failed to parse user id", zap.String("identity", participantIdentity), zap.Error(err))
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
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
func (h *WebhookHandler) handleParticipantLeftV2(c echo.Context, event *livekit.WebhookEvent) error {
	c.Logger().Info("🔹 [WEBHOOK] Processing participant_left")

	if event.Participant == nil || event.Room == nil {
		h.logger.Warn("participant or room missing in event")
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
	}

	participantIdentity := event.Participant.Identity
	roomName := event.Room.Name

	c.Logger().Infof("👋 [WEBHOOK] Participant left: %s from room %s", participantIdentity, roomName)

	userID, err := uuid.Parse(participantIdentity)
	if err != nil {
		h.logger.Error("failed to parse user id", zap.String("identity", participantIdentity), zap.Error(err))
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
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
func (h *WebhookHandler) handleRoomStartedV2(c echo.Context, event *livekit.WebhookEvent) error {
	c.Logger().Info("🔹 [WEBHOOK] Processing room_started")

	if event.Room == nil {
		h.logger.Warn("room missing in event")
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
	}

	roomName := event.Room.Name
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
func (h *WebhookHandler) handleRoomFinishedV2(c echo.Context, event *livekit.WebhookEvent) error {
	c.Logger().Info("🔹 [WEBHOOK] Processing room_finished")

	if event.Room == nil {
		h.logger.Warn("room missing in event")
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
	}

	roomName := event.Room.Name
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

	h.logger.Info("room finished - waiting for egress recording", zap.String("room_id", roomEntity.ID.String()))

	// Recording will be handled by egress_ended webhook when egress completes
	// No need to create fake recording here - LiveKit will auto record via egress

	return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok", "event": "room_finished"})
}

// handleEgressEndedV2 handles egress_ended event (recording completed)
func (h *WebhookHandler) handleEgressEndedV2(c echo.Context, event *livekit.WebhookEvent) error {
	c.Logger().Info("🔹 [WEBHOOK] Processing egress_ended")

	if event.EgressInfo == nil {
		h.logger.Warn("❌ egress info missing in event")
		c.Logger().Warn("egress info missing in event")
		return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
	}

	egressID := event.EgressInfo.EgressId
	roomName := event.EgressInfo.RoomName

	c.Logger().Infof("🎬 Egress ended: %s (room: %s)", egressID, roomName)

	// Get context early for MinIO operations
	ctx := c.Request().Context()

	// Extract recording URL from FileResults
	// LiveKit creates separate audio and video track files
	// AssemblyAI needs audio files only (.ogg, .mp3, .wav, etc.)
	var recordingURL string
	var filename string

	if len(event.EgressInfo.FileResults) > 0 {
		// Find the audio file from the results
		// Audio files typically have extensions: .ogg, .mp3, .wav, .m4a
		var audioFilename string
		for _, fileResult := range event.EgressInfo.FileResults {
			fname := fileResult.Filename
			// Check if it's an audio file
			if strings.HasSuffix(strings.ToLower(fname), ".ogg") ||
				strings.HasSuffix(strings.ToLower(fname), ".mp3") ||
				strings.HasSuffix(strings.ToLower(fname), ".wav") ||
				strings.HasSuffix(strings.ToLower(fname), ".m4a") ||
				strings.Contains(strings.ToLower(fname), "audio-") {
				audioFilename = fname
				break
			}
		}

		// If no audio file found, skip this egress event
		if audioFilename == "" {
			h.logger.Info("⏭️  No audio file in this egress, skipping AssemblyAI submission (likely video-only egress)",
				zap.Int("file_count", len(event.EgressInfo.FileResults)))
			return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok", "event": "egress_ended_no_audio"})
		}

		filename = audioFilename

		h.logger.Info("🔍 Audio file found in egress results",
			zap.String("filename", filename),
			zap.Int("total_files", len(event.EgressInfo.FileResults)))

		if filename != "" {
			// Generate presigned URL from MinIO for AssemblyAI access
			// URL expires in 24 hours (enough time for AssemblyAI to download and process)
			if h.minioClient != nil {
				presignedURL, err := h.minioClient.GetFileURL(ctx, filename, 24*time.Hour)
				if err != nil {
					h.logger.Error("❌ Failed to generate presigned URL",
						zap.String("filename", filename),
						zap.Error(err))
				} else {
					recordingURL = presignedURL
					h.logger.Info("✅ Generated presigned URL for audio file",
						zap.String("filename", filename),
						zap.String("presigned_url", recordingURL))
				}
			} else {
				h.logger.Warn("⚠️ MinIO client not available, using S3 path")
				bucketName := os.Getenv("MINIO_BUCKET_NAME")
				if bucketName == "" {
					bucketName = "meeting-recordings"
				}
				recordingURL = "s3://" + bucketName + "/" + filename
			}
		} else {
			h.logger.Warn("⚠️ FileResult filename is empty")
		}
	} else {
		h.logger.Warn("⚠️ No files in egress results")
	}

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

	// Submit to AssemblyAI for transcription
	if err := h.aiService.SubmitToAssemblyAI(ctx, roomEntity.ID, recordingURL); err != nil {
		h.logger.Error("❌ failed to submit to AssemblyAI",
			zap.String("room_id", roomEntity.ID.String()),
			zap.String("recording_url", recordingURL),
			zap.Error(err))
	} else {
		h.logger.Info("✅ Successfully submitted to AssemblyAI",
			zap.String("room_id", roomEntity.ID.String()))
	}

	return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok", "event": "egress_ended"})
}
