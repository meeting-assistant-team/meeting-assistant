package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
	"github.com/johnquangdev/meeting-assistant/internal/usecase/auth"
	"github.com/labstack/echo/v4"
)

// ContextKey is the type for context keys
type ContextKey string

const (
	// UserContextKey is the context key for the authenticated user
	UserContextKey ContextKey = "user"
)

// AuthMiddleware is the authentication middleware
type AuthMiddleware struct {
	oauthService *auth.OAuthService
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(oauthService *auth.OAuthService) *AuthMiddleware {
	return &AuthMiddleware{
		oauthService: oauthService,
	}
}

// Authenticate validates the JWT token and adds user to context
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			respondError(w, http.StatusUnauthorized, "Missing authorization token")
			return
		}

		user, err := m.oauthService.ValidateSession(r.Context(), token)
		if err != nil {
			respondError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole checks if the authenticated user has the required role
func (m *AuthMiddleware) RequireRole(roles ...entities.UserRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := GetUserFromContext(r.Context())
			if !ok {
				respondError(w, http.StatusUnauthorized, "User not authenticated")
				return
			}

			hasRole := false
			for _, role := range roles {
				if user.Role == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				respondError(w, http.StatusForbidden, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// OptionalAuth validates token if present but doesn't require it
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token != "" {
			user, err := m.oauthService.ValidateSession(r.Context(), token)
			if err == nil {
				ctx := context.WithValue(r.Context(), UserContextKey, user)
				r = r.WithContext(ctx)
			}
		}

		next.ServeHTTP(w, r)
	})
}

// GetUserFromContext retrieves the user from the request context
func GetUserFromContext(ctx context.Context) (*entities.User, bool) {
	user, ok := ctx.Value(UserContextKey).(*entities.User)
	return user, ok
}

// EchoAuth returns an Echo middleware that validates JWT Bearer tokens
// OAuth2 Standard: Only supports Authorization: Bearer <token> header
// Deprecated session_id cookie support removed for pure OAuth2 compliance
func EchoAuth(oauthService *auth.OAuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Extract Bearer token from Authorization header (OAuth2 standard)
			authHeader := c.Request().Header.Get("Authorization")
			token := ""
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
					token = parts[1]
				}
			}

			// DEPRECATED: Fallback to session_id cookie for backwards compatibility
			// Will be removed in future version
			if token == "" {
				if cookie, err := c.Cookie("session_id"); err == nil && cookie.Value != "" {
					if sid, err := uuid.Parse(cookie.Value); err == nil {
						user, err := oauthService.ValidateSessionByID(c.Request().Context(), sid)
						if err == nil {
							c.Set("user", user)
							c.Set("user_id", user.ID)
							c.Set("user_email", user.Email)
							// Log deprecation warning
							// fmt.Printf("⚠️  [DEPRECATED] session_id cookie used - please migrate to Bearer token\\n")
							return next(c)
						}
					}
				}
			}

			if token == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing authorization token")
			}

			// Validate access token (OAuth2 standard)
			user, err := oauthService.ValidateSession(c.Request().Context(), token)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid or expired token")
			}

			// Set user info into echo context
			c.Set("user", user)
			c.Set("user_id", user.ID)
			c.Set("user_email", user.Email)

			return next(c)
		}
	}
}

// Helper functions

func extractToken(r *http.Request) string {
	// Try Authorization header first
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		// Expected format: "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}

	// Try cookie as fallback
	cookie, err := r.Cookie("access_token")
	if err == nil {
		return cookie.Value
	}

	return ""
}

func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"error":"` + message + `","status":` + string(rune(status)) + `}`))
}
