package handler

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/johnquangdev/meeting-assistant/errors"
	_ "github.com/johnquangdev/meeting-assistant/internal/adapter/dto/auth" // for swagger
	"github.com/johnquangdev/meeting-assistant/internal/adapter/presenter"
	authUsecase "github.com/johnquangdev/meeting-assistant/internal/usecase/auth"
	"github.com/johnquangdev/meeting-assistant/pkg/config"
)

// Auth handles authentication HTTP requests
type Auth struct {
	oauthService *authUsecase.OAuthService
	logger       *zap.Logger
	cfg          *config.Config
}

// NewAuth creates a new auth handler
func NewAuth(oauthService *authUsecase.OAuthService, logger *zap.Logger, cfg *config.Config) *Auth {
	return &Auth{
		oauthService: oauthService,
		logger:       logger,
		cfg:          cfg,
	}
}

// GoogleLogin handles the initial Google OAuth login request
// @Summary      Initiate Google OAuth login
// @Description  Redirects user to Google OAuth consent screen. State stored in-memory (15 min expiry) and as HttpOnly cookie (CSRF protection).
// @Tags         Authentication
// @Produce      json
// @Success      307  {string}  string  "Redirect to Google OAuth consent page"
// @Failure      500  {object}  map[string]interface{}  "Internal server error"
// @Router       /auth/google/login [get]
func (h *Auth) GoogleLogin(c echo.Context) error {
	ctx := c.Request().Context()

	authURL, err := h.oauthService.GetGoogleAuthURL(ctx)
	if err != nil {
		return HandleError(h.logger, c, errors.ErrInternal(err))
	}

	// State is stored in-memory by stateManager with 15 minute expiration
	// Also stored as HttpOnly cookie for additional CSRF protection verification
	if h.logger != nil {
		h.logger.Info("generated OAuth state token", zap.String("state_hash", authURL.State[:8]))
	}

	// Set state as HttpOnly cookie for CSRF verification during callback
	cookiePath := h.cfg.Server.CookiePath
	if cookiePath == "" {
		cookiePath = "/v1"
	}

	stateCookie := &http.Cookie{
		Name:     "oauth_state",
		Value:    authURL.State,
		Path:     cookiePath,
		Domain:   h.cfg.Server.CookieDomain,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
		MaxAge:   15 * 60, // 15 minutes, same as state expiration
	}
	c.SetCookie(stateCookie)

	// Redirect to Google OAuth
	return c.Redirect(http.StatusTemporaryRedirect, authURL.URL)
}

// GoogleCallback handles the OAuth callback from Google
// @Summary      Handle Google OAuth callback
// @Description  Processes the OAuth callback from Google and sets a HttpOnly session cookie. Redirects to frontend callback URL configured in FrontendURL setting.
// @Tags         Authentication
// @Produce      json
// @Param        code   query     string  true  "Authorization code from Google"
// @Param        state  query     string  true  "State parameter for CSRF protection"
// @Success      307    {string}  string  "Redirect to frontend callback URL with session_id HttpOnly cookie"
// @Failure      400    {object}  map[string]interface{}  "Missing code or state parameter"
// @Failure      401    {object}  map[string]interface{}  "Authentication failed - invalid code or state"
// @Router       /auth/google/callback [get]
func (h *Auth) GoogleCallback(c echo.Context) error {
	ctx := c.Request().Context()

	code := c.QueryParam("code")
	state := c.QueryParam("state")

	if code == "" || state == "" {
		return HandleError(h.logger, c, errors.ErrInvalidArgument("Missing code or state parameter"))
	}

	// Verify state cookie matches query parameter (CSRF double-check)
	// This is a lightweight check before calling the service
	stateCookie, err := c.Cookie("oauth_state")
	if err == nil && stateCookie != nil && stateCookie.Value != "" {
		if stateCookie.Value != state {
			if h.logger != nil {
				h.logger.Warn("state mismatch between cookie and query parameter",
					zap.String("state_hash", state[:8]))
			}
			return HandleError(h.logger, c, errors.ErrUnauthenticated().WithDetail("error", "state parameter mismatch - CSRF validation failed"))
		}
	}

	// Note: State validation (one-time use check) is done in HandleGoogleCallback service method
	// to avoid double validation that would consume the state token

	req := &authUsecase.GoogleCallbackRequest{
		Code:  code,
		State: state,
	}

	usecaseResp, err := h.oauthService.HandleGoogleCallback(ctx, req)
	if err != nil {
		return HandleError(h.logger, c, errors.ErrUnauthenticated().WithDetail("error", err.Error()))
	}

	// Create a server-side session cookie (store session ID only, session/refresh token already saved server-side)
	sessionID := usecaseResp.SessionID
	if sessionID == "" {
		return HandleError(h.logger, c, errors.ErrInternal(fmt.Errorf("missing session id")))
	}

	cookieDomain := h.cfg.Server.CookieDomain
	cookiePath := h.cfg.Server.CookiePath
	if cookiePath == "" {
		cookiePath = "/v1"
	}

	// Cookie expiry equals refresh token expiry
	sessionMaxAge := int(h.cfg.JWT.RefreshExpiry.Seconds())
	if sessionMaxAge <= 0 {
		sessionMaxAge = 7 * 24 * 60 * 60
	}

	sessionCookie := &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     cookiePath,
		Domain:   cookieDomain,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
		MaxAge:   sessionMaxAge,
	}

	c.SetCookie(sessionCookie)

	// Clear oauth_state cookie (one-time use)
	stateClearCookie := &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     cookiePath,
		Domain:   cookieDomain,
		HttpOnly: true,
		Secure:   true,
		MaxAge:   -1,
	}
	c.SetCookie(stateClearCookie)

	// Redirect to frontend callback URL from config
	redirectTarget := h.cfg.Server.FrontendURL + "/auth/callback"
	if h.logger != nil {
		h.logger.Info("redirecting to frontend callback",
			zap.String("redirect_url", redirectTarget))
	}
	return c.Redirect(http.StatusTemporaryRedirect, redirectTarget)
}

// RefreshToken refreshes the access token
// @Summary      Refresh access token
// @Description  Gets a new access token using session_id from HttpOnly cookie. No request body needed.
// @Tags         Authentication
// @Produce      json
// @Success      200      {object}  map[string]interface{}  "Token refreshed successfully with access_token and expires_in"
// @Failure      400      {object}  map[string]interface{}  "Invalid or missing session_id cookie"
// @Failure      401      {object}  map[string]interface{}  "Failed to refresh token"
// @Router       /auth/refresh [post]
func (h *Auth) RefreshToken(c echo.Context) error {
	ctx := c.Request().Context()

	// Read session_id from HttpOnly cookie
	cookie, err := c.Cookie("session_id")
	if err != nil || cookie == nil || cookie.Value == "" {
		return HandleError(h.logger, c, errors.ErrInvalidToken())
	}

	sid, err := uuid.Parse(cookie.Value)
	if err != nil {
		return HandleError(h.logger, c, errors.ErrInvalidToken())
	}

	usecaseResp, err := h.oauthService.RefreshAccessTokenBySessionID(ctx, sid)
	if err != nil {
		return HandleError(h.logger, c, err)
	}

	// Return access token JSON (no refresh token)
	data := map[string]interface{}{
		"access_token": usecaseResp.AccessToken,
		"expires_in":   int(usecaseResp.ExpiresIn),
	}
	return HandleSuccess(h.logger, c, data)
}

// Logout logs out the current user
// @Summary      Logout user
// @Description  Invalidates the session and logs out the user. Supports session_id cookie (preferred) or refresh_token in body (backwards compatibility).
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request  body      object{refresh_token=string}  false  "Refresh token (optional, only for backwards compatibility)"
// @Success      200      {object}  map[string]string  "Logged out successfully"
// @Failure      400      {object}  map[string]interface{}  "Missing session or invalid token"
// @Failure      500      {object}  map[string]interface{}  "Failed to logout"
// @Router       /auth/logout [post]
func (h *Auth) Logout(c echo.Context) error {
	ctx := c.Request().Context()

	// Read session_id cookie
	cookie, err := c.Cookie("session_id")
	var sessionID uuid.UUID
	if err == nil {
		sessionID, err = uuid.Parse(cookie.Value)
		if err != nil {
			return HandleError(h.logger, c, errors.ErrInvalidArgument("Invalid session id cookie"))
		}
	} else {
		// Fallback to body (backwards compatibility)
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := c.Bind(&req); err != nil || req.RefreshToken == "" {
			return HandleError(h.logger, c, errors.ErrInvalidArgument("Missing session or refresh token"))
		}
		// Call existing logout by refresh token
		if err := h.oauthService.Logout(ctx, req.RefreshToken); err != nil {
			return HandleError(h.logger, c, errors.ErrInternal(err))
		}

		return HandleSuccess(h.logger, c, map[string]string{"message": "Logged out successfully"})
	}

	// Revoke session by ID
	if err := h.oauthService.RevokeSessionByID(ctx, sessionID); err != nil {
		return HandleError(h.logger, c, errors.ErrInternal(err))
	}

	// Clear session cookie
	cookiePath := h.cfg.Server.CookiePath
	if cookiePath == "" {
		cookiePath = "/"
	}
	clear := &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     cookiePath,
		Domain:   h.cfg.Server.CookieDomain,
		HttpOnly: true,
		Secure:   true,
		MaxAge:   -1,
	}
	c.SetCookie(clear)

	return HandleSuccess(h.logger, c, map[string]string{"message": "Logged out successfully"})
}

// Me returns the current user information
// @Summary      Get current user
// @Description  Returns the authenticated user's information. Supports Authorization header (Bearer) or session cookie.
// @Tags         Authentication
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  github_com_johnquangdev_meeting-assistant_internal_adapter_dto_auth.UserResponse  "User information"
// @Failure      401  {object}  map[string]interface{}  "Missing or invalid token/session"
// @Router       /auth/me [get]
func (h *Auth) Me(c echo.Context) error {
	ctx := c.Request().Context()

	// Extract token from Authorization header
	token := c.Request().Header.Get("Authorization")
	if token == "" {
		// Try to get session cookie
		if cookie, err := c.Cookie("session_id"); err == nil {
			if cookie.Value != "" {
				if sid, err := uuid.Parse(cookie.Value); err == nil {
					user, err := h.oauthService.ValidateSessionByID(ctx, sid)
					if err != nil {
						return HandleError(h.logger, c, errors.ErrUnauthenticated().WithDetail("error", err.Error()))
					}
					response := presenter.ToUserResponse(user)
					return HandleSuccess(h.logger, c, response)
				}
			}
		}
	} else {
		// Remove "Bearer " prefix if present
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		user, err := h.oauthService.ValidateSession(ctx, token)
		if err != nil {
			return HandleError(h.logger, c, errors.ErrUnauthenticated().WithDetail("error", err.Error()))
		}

		response := presenter.ToUserResponse(user)
		return HandleSuccess(h.logger, c, response)
	}

	return HandleError(h.logger, c, errors.ErrUnauthenticated().WithDetail("error", "Missing authorization token or session"))
}

// TestToken generates a test JWT token for development (DO NOT USE IN PRODUCTION)
// @Summary      Generate test JWT token (Development Only)
// @Description  Generates a test JWT token for testing API with Postman. ONLY for development/testing. Use OAuth flow in production.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request  body      object{email=string}  false  "User email (optional, default: test@example.com)"
// @Success      200      {object}  map[string]interface{}  "access_token, expires_in"
// @Router       /auth/test/token [post]
func (h *Auth) TestToken(c echo.Context) error {
	// Only allow in development mode
	if h.cfg.Server.Environment != "development" {
		return HandleError(h.logger, c, errors.ErrForbidden("Test token endpoint only available in development"))
	}

	// Parse request body for email
	var req struct {
		Email string `json:"email"`
	}
	if err := c.Bind(&req); err != nil {
		// If body parsing fails, use default email
		req.Email = "test@example.com"
	}
	// Generate test user ID
	testUserID := uuid.New()

	// Create JWT token using jwtManager (via oauth service)
	ctx := c.Request().Context()
	resp, err := h.oauthService.CreateTestAccessToken(ctx, testUserID, req.Email)
	if err != nil {
		return HandleError(h.logger, c, errors.ErrInternal(err))
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"access_token": resp.AccessToken,
		"expires_in":   resp.ExpiresIn,
		"email":        req.Email,
		"message":      "Test token generated - USE FOR TESTING ONLY",
	})
}
