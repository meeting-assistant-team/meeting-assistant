package entities

import (
	"time"

	"github.com/google/uuid"
)

// Session represents a user session
type Session struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID       uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index"`
	RefreshToken string     `json:"-" gorm:"column:refresh_token;type:text;uniqueIndex;not null"` // Store plain refresh token
	CreatedAt    time.Time  `json:"created_at" gorm:"autoCreateTime"`
	ExpiresAt    time.Time  `json:"expires_at" gorm:"type:timestamp;not null;index"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty" gorm:"type:timestamp"`
	LastUsedAt   *time.Time `json:"last_used_at,omitempty" gorm:"type:timestamp"`

	// Device info
	DeviceInfo map[string]interface{} `json:"device_info,omitempty" gorm:"type:jsonb"`
	IPAddress  *string                `json:"ip_address,omitempty" gorm:"type:varchar(45)"`
	UserAgent  *string                `json:"user_agent,omitempty" gorm:"type:text"`
}

// NewSession creates a new session
func NewSession(userID uuid.UUID, refreshToken string, expiresAt time.Time) *Session {
	return &Session{
		ID:           uuid.New(),
		UserID:       userID,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		RevokedAt:    nil,
		CreatedAt:    time.Now(),
	}
}

// IsExpired checks if session is expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsValid checks if session is valid (not expired and not revoked)
func (s *Session) IsValid() bool {
	if s == nil {
		return false
	}
	return !s.IsExpired() && s.RevokedAt == nil
}

// Revoke revokes the session
func (s *Session) Revoke() {
	now := time.Now()
	s.RevokedAt = &now
}

// UpdateLastUsed updates the last used timestamp
func (s *Session) UpdateLastUsed() {
	now := time.Now()
	s.LastUsedAt = &now
}

// WithDeviceInfo adds device information
func (s *Session) WithDeviceInfo(ip, userAgent string) *Session {
	s.IPAddress = &ip
	s.UserAgent = &userAgent
	return s
}
