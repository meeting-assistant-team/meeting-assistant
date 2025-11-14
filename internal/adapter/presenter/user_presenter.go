package presenter

import (
	"encoding/json"

	authDTO "github.com/johnquangdev/meeting-assistant/internal/adapter/dto/auth"
	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
	"github.com/johnquangdev/meeting-assistant/internal/usecase/auth"
)

// ToUserResponse converts a User entity to UserResponse DTO
func ToUserResponse(u *entities.User) *authDTO.UserResponse {
	if u == nil {
		return nil
	}

	// Parse preferences from JSON
	var notificationPrefs, meetingPrefs map[string]interface{}
	if u.NotificationPreferences != nil {
		json.Unmarshal(u.NotificationPreferences, &notificationPrefs)
	}
	if u.MeetingPreferences != nil {
		json.Unmarshal(u.MeetingPreferences, &meetingPrefs)
	}

	response := &authDTO.UserResponse{
		ID:                      u.ID.String(),
		Email:                   u.Email,
		Name:                    u.Name,
		OAuthProvider:           "",
		EmailVerified:           false,
		NotificationPreferences: notificationPrefs,
		MeetingPreferences:      meetingPrefs,
		LastLoginAt:             u.LastLoginAt,
		CreatedAt:               u.CreatedAt,
		UpdatedAt:               u.UpdatedAt,
	}

	// Set optional fields
	if u.AvatarURL != nil {
		response.AvatarURL = *u.AvatarURL
	}
	if u.OAuthProvider != nil {
		response.OAuthProvider = *u.OAuthProvider
	}

	return response
}

// ToAuthResponse converts usecase AuthResponse to DTO AuthResponse
// ToAuthRefreshTokenResponse converts usecase AuthResponse to DTO RefreshTokenResponse (for refresh endpoint)
func ToAuthRefreshTokenResponse(usecaseResp *auth.AuthResponse) *authDTO.RefreshTokenResponse {
	if usecaseResp == nil {
		return nil
	}
	return &authDTO.RefreshTokenResponse{
		AccessToken: usecaseResp.AccessToken,
		ExpiresIn:   int(usecaseResp.ExpiresIn),
		TokenType:   "Bearer",
	}
}

// ToAuthResponse converts usecase AuthResponse to DTO AuthResponse
func ToAuthResponse(usecaseResp *auth.AuthResponse) *authDTO.AuthResponse {
	if usecaseResp == nil {
		return nil
	}

	return &authDTO.AuthResponse{
		AccessToken:  usecaseResp.AccessToken,
		RefreshToken: usecaseResp.RefreshToken,
		ExpiresIn:    int(usecaseResp.ExpiresIn),
		TokenType:    "Bearer",
		User:         ToUserResponse(usecaseResp.User),
	}
}
