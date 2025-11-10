package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/johnquangdev/meeting-assistant/internal/usecase/auth"
)

// Auth handles authentication HTTP requests
type Auth struct {
	oauthService *auth.OAuthService
}

// NewAuth creates a new auth handler
func NewAuth(oauthService *auth.OAuthService) *Auth {
	return &Auth{
		oauthService: oauthService,
	}
}

// GoogleLogin handles the initial Google OAuth login request
// GET /api/v1/auth/google/login
func (h *Auth) GoogleLogin(c echo.Context) error {
	ctx := c.Request().Context()

	authURL, err := h.oauthService.GetGoogleAuthURL(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error":   "Failed to generate auth URL",
			"details": err.Error(),
		})
	}

	// Redirect to Google OAuth
	return c.Redirect(http.StatusTemporaryRedirect, authURL.URL)
}

// GoogleCallback handles the OAuth callback from Google
// GET /api/v1/auth/google/callback
func (h *Auth) GoogleCallback(c echo.Context) error {
	ctx := c.Request().Context()

	code := c.QueryParam("code")
	state := c.QueryParam("state")

	if code == "" || state == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Missing code or state parameter",
		})
	}

	req := &auth.GoogleCallbackRequest{
		Code:  code,
		State: state,
	}

	response, err := h.oauthService.HandleGoogleCallback(ctx, req)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error":   "Authentication failed",
			"details": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, response)
}

// RefreshToken refreshes the access token
// POST /api/v1/auth/refresh
func (h *Auth) RefreshToken(c echo.Context) error {
	ctx := c.Request().Context()

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
	}

	if req.RefreshToken == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Missing refresh token",
		})
	}

	response, err := h.oauthService.RefreshAccessToken(ctx, req.RefreshToken)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error":   "Failed to refresh token",
			"details": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, response)
}

// Logout logs out the current user
// POST /api/v1/auth/logout
func (h *Auth) Logout(c echo.Context) error {
	ctx := c.Request().Context()

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := c.Bind(&req); err != nil || req.RefreshToken == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Missing refresh token",
		})
	}

	if err := h.oauthService.Logout(ctx, req.RefreshToken); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error":   "Failed to logout",
			"details": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Logged out successfully",
	})
}

// Me returns the current user information
// GET /api/v1/auth/me
func (h *Auth) Me(c echo.Context) error {
	ctx := c.Request().Context()

	// Extract token from Authorization header
	token := c.Request().Header.Get("Authorization")
	if token == "" {
		// Try to get from cookie
		cookie, err := c.Cookie("access_token")
		if err == nil {
			token = cookie.Value
		}
	} else {
		// Remove "Bearer " prefix if present
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
	}

	if token == "" {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error": "Missing authorization token",
		})
	}

	user, err := h.oauthService.ValidateSession(ctx, token)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error":   "Invalid session",
			"details": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"user": user.ToPublic(),
	})
}
