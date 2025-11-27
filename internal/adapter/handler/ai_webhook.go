package handler

import (
	"io"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/johnquangdev/meeting-assistant/errors"
	aiuse "github.com/johnquangdev/meeting-assistant/internal/usecase/ai"
)

// AIWebhookHandler handles incoming webhooks from AI providers (AssemblyAI)
type AIWebhookHandler struct {
	svc    aiuse.Service
	secret string
	logger *zap.Logger
}

// NewAIWebhookHandler creates a new handler
func NewAIWebhookHandler(svc aiuse.Service, secret string, logger *zap.Logger) *AIWebhookHandler {
	return &AIWebhookHandler{svc: svc, secret: secret, logger: logger}
}

// HandleAssemblyAIWebhook receives webhooks from AssemblyAI
func (h *AIWebhookHandler) HandleAssemblyAIWebhook(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return HandleError(h.logger, c, errors.ErrInvalidPayload())
	}

	// AssemblyAI signs requests in a header; try common header names
	signature := c.Request().Header.Get("x-assemblyai-signature")
	if signature == "" {
		signature = c.Request().Header.Get("Authorization")
	}

	if err := h.svc.HandleAssemblyAIWebhook(c.Request().Context(), body, signature); err != nil {
		if h.logger != nil {
			h.logger.Error("ai webhook handler error", zap.Error(err))
		}
		return HandleError(h.logger, c, errors.ErrProcessingFailed(err))
	}
	return HandleSuccess(h.logger, c, map[string]interface{}{"status": "ok"})
}
