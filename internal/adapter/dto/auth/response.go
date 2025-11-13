package auth

import "time"

// UserResponse represents user information in responses
type UserResponse struct {
	ID                      string                 `json:"id"`
	Email                   string                 `json:"email"`
	Name                    string                 `json:"name"`
	AvatarURL               string                 `json:"avatar_url,omitempty"`
	OAuthProvider           string                 `json:"oauth_provider"`
	EmailVerified           bool                   `json:"email_verified"`
	NotificationPreferences map[string]interface{} `json:"notification_preferences,omitempty"`
	MeetingPreferences      map[string]interface{} `json:"meeting_preferences,omitempty"`
	LastLoginAt             *time.Time             `json:"last_login_at,omitempty"`
	CreatedAt               time.Time              `json:"created_at"`
	UpdatedAt               time.Time              `json:"updated_at"`
}

// AuthResponse represents the authentication response with tokens
type AuthResponse struct {
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	ExpiresIn    int           `json:"expires_in"` // seconds
	TokenType    string        `json:"token_type"` // "Bearer"
	User         *UserResponse `json:"user"`
}

// RefreshTokenResponse represents the response after refreshing token
type RefreshTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}
