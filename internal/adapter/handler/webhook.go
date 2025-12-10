package handler

import (
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	aiUsecase "github.com/johnquangdev/meeting-assistant/internal/usecase/ai"
	roomUsecase "github.com/johnquangdev/meeting-assistant/internal/usecase/room"
)

// WebhookHandler handles LiveKit webhook events
type WebhookHandler struct {
	roomService   roomUsecase.Service
	aiService     aiUsecase.Service
	livekitAPIKey string
	livekitSecret string
	logger        *zap.Logger
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(roomService roomUsecase.Service, aiService aiUsecase.Service, livekitAPIKey string, livekitSecret string, logger *zap.Logger) *WebhookHandler {
	return &WebhookHandler{
		roomService:   roomService,
		aiService:     aiService,
		livekitAPIKey: livekitAPIKey,
		livekitSecret: livekitSecret,
		logger:        logger,
	}
}

// HandleLiveKitWebhook redirects to v2 implementation with proper JWT signature validation
// @Summary      LiveKit Webhook
// @Description  Receives webhook events from LiveKit server with JWT signature validation
// @Tags         Webhooks
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /webhooks/livekit [post]
func (h *WebhookHandler) HandleLiveKitWebhook(c echo.Context) error {
	return h.HandleLiveKitWebhookV2(c)
}
