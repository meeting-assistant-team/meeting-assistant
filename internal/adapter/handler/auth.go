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

	return c.JSON(http.StatusOK, map[string]interface{}{
		"auth_url":   authURL.URL,
		"state":      authURL.State,
		"expires_in": 900, // 15 minutes
	})
}

// GoogleCallback handles the OAuth callback from Google
// @Summary      Handle Google OAuth callback
// @Description  Processes the OAuth callback from Google. Returns tokens (OAuth2 RFC 6749 compliant) and sets HttpOnly cookies.
// @Tags         Authentication
// @Produce      json
// @Param        code   query     string  true  "Authorization code from Google"
// @Param        state  query     string  true  "State parameter for CSRF protection"
// @Success      200    {object}  github_com_johnquangdev_meeting-assistant_internal_adapter_dto_auth.AuthResponse  "OAuth2 tokens with user info"
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

	// OAuth2 Standard: Set refresh_token as HttpOnly cookie (secure storage)
	refreshToken := usecaseResp.RefreshToken
	if refreshToken == "" {
		return HandleError(h.logger, c, errors.ErrInternal(fmt.Errorf("missing refresh token")))
	}

	cookieDomain := h.cfg.Server.CookieDomain
	cookiePath := h.cfg.Server.CookiePath
	if cookiePath == "" {
		cookiePath = "/v1"
	}

	// Refresh token cookie (7 days by default)
	refreshMaxAge := int(h.cfg.JWT.RefreshExpiry.Seconds())
	if refreshMaxAge <= 0 {
		refreshMaxAge = 7 * 24 * 60 * 60
	}

	refreshCookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     cookiePath,
		Domain:   cookieDomain,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode, // Lax for OAuth2 callback
		MaxAge:   refreshMaxAge,
	}
	c.SetCookie(refreshCookie)

	// DEPRECATED: Also set session_id cookie for backwards compatibility
	if usecaseResp.SessionID != "" {
		sessionCookie := &http.Cookie{
			Name:     "session_id",
			Value:    usecaseResp.SessionID,
			Path:     cookiePath,
			Domain:   cookieDomain,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteNoneMode,
			MaxAge:   refreshMaxAge,
		}
		c.SetCookie(sessionCookie)
	}

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

	// ✅ RFC 6749 OAuth2 Standard: Return JSON response with tokens
	// Frontend will handle navigation to callback URL based on response
	accessExpiry := int(h.cfg.JWT.AccessExpiry.Seconds())

	authResp := presenter.ToAuthResponse(usecaseResp)

	if h.logger != nil {
		h.logger.Info("OAuth2 callback successful",
			zap.String("user_id", usecaseResp.User.ID.String()),
			zap.String("email", usecaseResp.User.Email))
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"access_token":  authResp.AccessToken,
		"refresh_token": authResp.RefreshToken,
		"expires_in":    accessExpiry,
		"token_type":    "Bearer",
		"user": map[string]interface{}{
			"id":                usecaseResp.User.ID,
			"email":             usecaseResp.User.Email,
			"name":              usecaseResp.User.Name,
			"role":              usecaseResp.User.Role,
			"avatar_url":        usecaseResp.User.AvatarURL,
			"is_email_verified": usecaseResp.User.IsEmailVerified,
		},
		"callback_url": h.cfg.Server.FrontendURL + "/auth/callback",
	})
}

// RefreshToken refreshes the access token
// @Summary      Refresh access token
// @Description  Gets new access and refresh tokens using refresh_token from cookie or request body. Implements OAuth2 token rotation.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request  body      object{refresh_token=string}  false  "Refresh token (optional if cookie present)"
// @Success      200      {object}  github_com_johnquangdev_meeting-assistant_internal_adapter_dto_auth.RefreshTokenResponse  "New tokens with rotation"
// @Failure      400      {object}  map[string]interface{}  "Invalid or missing refresh_token"
// @Failure      401      {object}  map[string]interface{}  "Failed to refresh token"
// @Router       /auth/refresh [post]
func (h *Auth) RefreshToken(c echo.Context) error {
	ctx := c.Request().Context()

	var refreshToken string

	// Try to get refresh_token from HttpOnly cookie first (OAuth2 recommended)
	cookie, err := c.Cookie("refresh_token")
	if err == nil && cookie != nil && cookie.Value != "" {
		refreshToken = cookie.Value
	} else {
		// Fallback to request body (for clients that can't use cookies)
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := c.Bind(&req); err == nil && req.RefreshToken != "" {
			refreshToken = req.RefreshToken
		}
	}

	// If still no refresh_token, try deprecated session_id (backwards compatibility)
	if refreshToken == "" {
		var sessionIDValue string
		sessionIDValue = c.Request().Header.Get("session_id")
		if sessionIDValue == "" {
			cookie, err := c.Cookie("session_id")
			if err == nil && cookie != nil && cookie.Value != "" {
				sessionIDValue = cookie.Value
			}
		}
		if sessionIDValue != "" {
			sid, err := uuid.Parse(sessionIDValue)
			if err == nil {
				usecaseResp, err := h.oauthService.RefreshAccessTokenBySessionID(ctx, sid)
				if err != nil {
					return HandleError(h.logger, c, err)
				}
				// Legacy response (no refresh_token)
				data := map[string]interface{}{
					"access_token": usecaseResp.AccessToken,
					"expires_in":   int(usecaseResp.ExpiresIn),
				}
				return HandleSuccess(h.logger, c, data)
			}
		}
	}

	if refreshToken == "" {
		return HandleError(h.logger, c, errors.ErrInvalidToken())
	}

	// Call OAuth2 token refresh with rotation
	usecaseResp, err := h.oauthService.RefreshAccessToken(ctx, refreshToken)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("refresh token failed", zap.Error(err))
		}
		return HandleError(h.logger, c, err)
	}

	// Set new refresh_token as HttpOnly cookie (token rotation)
	if usecaseResp.RefreshToken != "" {
		cookiePath := h.cfg.Server.CookiePath
		if cookiePath == "" {
			cookiePath = "/v1"
		}
		refreshMaxAge := int(h.cfg.JWT.RefreshExpiry.Seconds())
		if refreshMaxAge <= 0 {
			refreshMaxAge = 7 * 24 * 60 * 60
		}

		refreshCookie := &http.Cookie{
			Name:     "refresh_token",
			Value:    usecaseResp.RefreshToken,
			Path:     cookiePath,
			Domain:   h.cfg.Server.CookieDomain,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   refreshMaxAge,
		}
		c.SetCookie(refreshCookie)
	}

	// Return OAuth2 standard response (both tokens in body + cookie)
	response := presenter.ToAuthRefreshTokenResponse(usecaseResp)
	return HandleSuccess(h.logger, c, response)
}

// Logout logs out the current user
// @Summary      Logout user
// @Description  Revokes refresh token and clears cookies. Supports refresh_token from cookie (preferred) or request body.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request  body      object{refresh_token=string}  false  "Refresh token (optional if cookie present)"
// @Success      200      {object}  map[string]string  "Logged out successfully"
// @Failure      400      {object}  map[string]interface{}  "Missing refresh token"
// @Failure      500      {object}  map[string]interface{}  "Failed to logout"
// @Router       /auth/logout [post]
func (h *Auth) Logout(c echo.Context) error {
	ctx := c.Request().Context()

	var refreshToken string

	// Try refresh_token cookie first
	cookie, err := c.Cookie("refresh_token")
	if err == nil && cookie != nil && cookie.Value != "" {
		refreshToken = cookie.Value
	} else {
		// Fallback to request body
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := c.Bind(&req); err == nil && req.RefreshToken != "" {
			refreshToken = req.RefreshToken
		}
	}

	// Fallback to deprecated session_id (backwards compatibility)
	if refreshToken == "" {
		cookie, err := c.Cookie("session_id")
		if err == nil && cookie != nil && cookie.Value != "" {
			sessionID, err := uuid.Parse(cookie.Value)
			if err == nil {
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
		}
	}

	if refreshToken == "" {
		return HandleError(h.logger, c, errors.ErrInvalidArgument("Missing refresh token"))
	}

	// Extract access token from Authorization header (if present)
	var accessToken string
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		accessToken = authHeader[7:]
	}

	// Revoke token family and blacklist access token (OAuth2 standard)
	if err := h.oauthService.Logout(ctx, refreshToken, accessToken); err != nil {
		return HandleError(h.logger, c, errors.ErrInternal(err))
	}

	// Clear refresh_token cookie
	cookiePath := h.cfg.Server.CookiePath
	if cookiePath == "" {
		cookiePath = "/v1"
	}
	clear := &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     cookiePath,
		Domain:   h.cfg.Server.CookieDomain,
		HttpOnly: true,
		Secure:   true,
		MaxAge:   -1,
	}
	c.SetCookie(clear)

	// Also clear deprecated session_id cookie
	sessionClear := &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     cookiePath,
		Domain:   h.cfg.Server.CookieDomain,
		HttpOnly: true,
		Secure:   true,
		MaxAge:   -1,
	}
	c.SetCookie(sessionClear)

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
