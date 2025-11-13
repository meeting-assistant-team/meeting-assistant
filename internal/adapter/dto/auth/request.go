package auth

// RefreshTokenRequest represents the request to refresh access token
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// LogoutRequest represents the request to logout
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// UpdateProfileRequest represents the request to update user profile
type UpdateProfileRequest struct {
	Name                    *string                `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	AvatarURL               *string                `json:"avatar_url,omitempty" validate:"omitempty,url"`
	NotificationPreferences map[string]interface{} `json:"notification_preferences,omitempty"`
	MeetingPreferences      map[string]interface{} `json:"meeting_preferences,omitempty"`
}
