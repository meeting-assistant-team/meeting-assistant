package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
	"github.com/johnquangdev/meeting-assistant/internal/usecase/auth"
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
