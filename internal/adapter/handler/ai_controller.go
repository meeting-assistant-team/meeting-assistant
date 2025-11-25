package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/johnquangdev/meeting-assistant/errors"
	aiuse "github.com/johnquangdev/meeting-assistant/internal/usecase/ai"
)

// AIController handles API endpoints that trigger AI processing
type AIController struct {
	svc aiuse.Service
}

// NewAIController creates a new AI controller
func NewAIController(svc aiuse.Service) *AIController {
	return &AIController{svc: svc}
}

// ProcessMeeting triggers AI processing for a meeting
func (ac *AIController) ProcessMeeting(c echo.Context) error {
	meetingID := c.Param("id")
	var req struct {
		RecordingURL string `json:"recording_url"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errors.ErrInvalidPayload())
	}
	if req.RecordingURL == "" {
		return c.JSON(http.StatusBadRequest, errors.ErrMissingRecordingURL())
	}
	if err := ac.svc.StartProcessing(c.Request().Context(), meetingID, req.RecordingURL); err != nil {
		c.Logger().Errorf("failed to start processing: %v", err)
		return c.JSON(http.StatusInternalServerError, errors.ErrProcessingFailed(err))
	}
	return c.JSON(http.StatusAccepted, map[string]interface{}{"status": "processing_started"})
}
