package handler

import (
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/johnquangdev/meeting-assistant/errors"
	aiuse "github.com/johnquangdev/meeting-assistant/internal/usecase/ai"
)

// AIController handles API endpoints that trigger AI processing
type AIController struct {
	svc    aiuse.Service
	logger *zap.Logger
}

// NewAIController creates a new AI controller
func NewAIController(svc aiuse.Service, logger *zap.Logger) *AIController {
	return &AIController{svc: svc, logger: logger}
}

// ProcessMeeting triggers AI processing for a meeting
// @Summary      Process meeting recording
// @Description  Manually triggers AI processing (transcription and analysis) for a meeting recording
// @Tags         AI
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string                      true  "Meeting ID (UUID)"
// @Param        request  body      object{recording_url=string}  true  "Recording URL for processing"
// @Success      202      {object}  map[string]interface{}      "Processing started"
// @Failure      400      {object}  map[string]interface{}      "Missing recording_url or invalid meeting ID"
// @Failure      401      {object}  map[string]interface{}      "User not authenticated"
// @Failure      500      {object}  map[string]interface{}      "Failed to start processing"
// @Router       /ai/meetings/{id}/process [post]
func (ac *AIController) ProcessMeeting(c echo.Context) error {
	meetingID := c.Param("id")
	var req struct {
		RecordingURL string `json:"recording_url"`
	}
	if err := c.Bind(&req); err != nil {
		return HandleError(ac.logger, c, errors.ErrInvalidPayload())
	}
	if req.RecordingURL == "" {
		return HandleError(ac.logger, c, errors.ErrMissingRecordingURL())
	}
	if err := ac.svc.StartProcessing(c.Request().Context(), meetingID, req.RecordingURL); err != nil {
		if ac.logger != nil {
			ac.logger.Error("failed to start processing", zap.Error(err))
		}
		return HandleError(ac.logger, c, errors.ErrProcessingFailed(err))
	}
	return HandleSuccess(ac.logger, c, map[string]interface{}{"status": "processing_started"})
}
