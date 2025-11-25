package handler

import (
	"io"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/johnquangdev/meeting-assistant/errors"
	aiuse "github.com/johnquangdev/meeting-assistant/internal/usecase/ai"
)

// AIWebhookHandler handles incoming webhooks from AI providers (AssemblyAI)
type AIWebhookHandler struct {
	svc    aiuse.Service
	secret string
}

// NewAIWebhookHandler creates a new handler
func NewAIWebhookHandler(svc aiuse.Service, secret string) *AIWebhookHandler {
	return &AIWebhookHandler{svc: svc, secret: secret}
}

// HandleAssemblyAIWebhook receives webhooks from AssemblyAI
func (h *AIWebhookHandler) HandleAssemblyAIWebhook(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errors.ErrInvalidPayload())
	}

	// AssemblyAI signs requests in a header; try common header names
	signature := c.Request().Header.Get("x-assemblyai-signature")
	if signature == "" {
		signature = c.Request().Header.Get("Authorization")
	}

	if err := h.svc.HandleAssemblyAIWebhook(c.Request().Context(), body, signature); err != nil {
		c.Logger().Errorf("ai webhook handler error: %v", err)
		return c.JSON(http.StatusBadRequest, errors.ErrProcessingFailed(err))
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"status": "ok"})
}
