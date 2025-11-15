package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	_ "github.com/johnquangdev/meeting-assistant/internal/adapter/dto/auth" // for swagger
	"github.com/johnquangdev/meeting-assistant/internal/adapter/presenter"
	authUsecase "github.com/johnquangdev/meeting-assistant/internal/usecase/auth"
)

// Auth handles authentication HTTP requests
type Auth struct {
	oauthService *authUsecase.OAuthService
}

// NewAuth creates a new auth handler
func NewAuth(oauthService *authUsecase.OAuthService) *Auth {
	return &Auth{
		oauthService: oauthService,
	}
}

// GoogleLogin handles the initial Google OAuth login request
// @Summary      Initiate Google OAuth login
// @Description  Redirects user to Google OAuth consent screen. **Flow cho FE:** 1. Gọi endpoint này từ browser (`window.location.href = 'https://api-meeting.infoquang.id.vn/v1/auth/google/login'`). 2. User được redirect đến Google để đăng nhập. 3. Sau khi đăng nhập thành công, Google redirect về `/auth/google/callback`. 4. Backend xử lý và redirect về FRONTEND_URL với tokens trong query params.
// @Tags         Authentication
// @Produce      json
// @Success      307  {string}  string  "Redirect to Google OAuth"
// @Failure      500  {object}  map[string]interface{}  "Internal server error"
// @Router       /auth/google/login [get]
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
// @Summary      Handle Google OAuth callback
// @Description  Processes the OAuth callback from Google and returns JWT tokens
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        code   query     string  true  "Authorization code from Google"
// @Param        state  query     string  true  "State parameter for CSRF protection"
// @Success      200    {object}  github_com_johnquangdev_meeting-assistant_internal_adapter_dto_auth.AuthResponse  "Authentication successful"
// @Failure      400    {object}  map[string]interface{}  "Missing code or state"
// @Failure      401    {object}  map[string]interface{}  "Authentication failed"
// @Router       /auth/google/callback [get]
func (h *Auth) GoogleCallback(c echo.Context) error {
	ctx := c.Request().Context()

	code := c.QueryParam("code")
	state := c.QueryParam("state")

	if code == "" || state == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Missing code or state parameter",
		})
	}

	req := &authUsecase.GoogleCallbackRequest{
		Code:  code,
		State: state,
	}

	usecaseResp, err := h.oauthService.HandleGoogleCallback(ctx, req)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error":   "Authentication failed",
			"details": err.Error(),
		})
	}

	// Convert usecase response to DTO
	response := presenter.ToAuthResponse(usecaseResp)
	return c.JSON(http.StatusOK, response)
}

// RefreshToken refreshes the access token
// @Summary      Refresh access token
// @Description  Gets a new access token using a refresh token. **Khi nào dùng:** Khi access_token hết hạn (401 Unauthorized) hoặc trước khi hết hạn để tránh gián đoạn UX. **Request body:** `{"refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."}`
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request  body      object{refresh_token=string}  true  "Refresh token"
// @Success      200      {object}  github_com_johnquangdev_meeting-assistant_internal_adapter_dto_auth.RefreshTokenResponse  "Token refreshed successfully"
// @Failure      400      {object}  map[string]interface{}  "Invalid request or missing token"
// @Failure      401      {object}  map[string]interface{}  "Failed to refresh token"
// @Router       /auth/refresh [post]
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

	usecaseResp, err := h.oauthService.RefreshAccessToken(ctx, req.RefreshToken)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]interface{}{
			"error":   "Failed to refresh token",
			"details": err.Error(),
		})
	}

	// Convert usecase response to DTO (no refresh token in response)
	response := presenter.ToAuthRefreshTokenResponse(usecaseResp)
	return c.JSON(http.StatusOK, response)
}

// Logout logs out the current user
// @Summary      Logout user
// @Description  Invalidates the refresh token and logs out the user. **Request body:** `{"refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."}` **Sau khi logout thành công, FE cần:** Xóa access_token và refresh_token khỏi localStorage/sessionStorage, sau đó redirect user về trang login.
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request  body      object{refresh_token=string}  true  "Refresh token to invalidate"
// @Success      200      {object}  map[string]string  "Logged out successfully"
// @Failure      400      {object}  map[string]interface{}  "Missing refresh token"
// @Failure      500      {object}  map[string]interface{}  "Failed to logout"
// @Router       /auth/logout [post]
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
// @Summary      Get current user
// @Description  Returns the authenticated user's information. **Yêu cầu:** Header `Authorization: Bearer <access_token>`. Không có tham số query/body. **Ví dụ:** `curl -H "Authorization: Bearer <token>" https://api-meeting.infoquang.id.vn/v1/auth/me`
// @Tags         Authentication
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  github_com_johnquangdev_meeting-assistant_internal_adapter_dto_auth.UserResponse  "User information"
// @Failure      401  {object}  map[string]interface{}  "Missing or invalid token"
// @Router       /auth/me [get]
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

	// Convert entity to DTO
	response := presenter.ToUserResponse(user)
	return c.JSON(http.StatusOK, response)
}
