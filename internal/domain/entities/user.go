package entities

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// User represents a user in the system
type User struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Email    string    `json:"email" gorm:"type:varchar(255);uniqueIndex;not null"`
	Name     string    `json:"name" gorm:"type:varchar(255);not null"`
	Role     UserRole  `json:"role" gorm:"type:varchar(50);default:'participant';not null"`
	IsActive bool      `json:"is_active" gorm:"default:true;not null"`

	// OAuth fields
	OAuthProvider     *string `json:"oauth_provider,omitempty" gorm:"column:oauth_provider;type:varchar(50);index:idx_oauth"`
	OAuthID           *string `json:"oauth_id,omitempty" gorm:"column:oauth_id;type:varchar(255);index:idx_oauth"`
	OAuthRefreshToken *string `json:"-" gorm:"column:oauth_refresh_token;type:text"` // Never expose in JSON
	PasswordHash      *string `json:"-" gorm:"column:password_hash;type:text"`       // Never expose in JSON

	// Profile
	AvatarURL *string `json:"avatar_url,omitempty" gorm:"type:varchar(500)"`
	Bio       *string `json:"bio,omitempty" gorm:"type:text"`
	Timezone  string  `json:"timezone" gorm:"type:varchar(50);default:'UTC';not null"`
	Language  string  `json:"language" gorm:"type:varchar(10);default:'en';not null"`

	// Status
	IsEmailVerified bool       `json:"is_email_verified" gorm:"default:false;not null"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty" gorm:"type:timestamp"`
	LastActiveAt    *time.Time `json:"last_active_at,omitempty" gorm:"type:timestamp"`

	// Preferences (stored as JSONB in PostgreSQL)
	NotificationPreferences datatypes.JSON `json:"notification_preferences" gorm:"type:jsonb;default:'{}'"`
	MeetingPreferences      datatypes.JSON `json:"meeting_preferences" gorm:"type:jsonb;default:'{}'"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// UserRole defines user roles
type UserRole string

const (
	RoleAdmin       UserRole = "admin"
	RoleHost        UserRole = "host"
	RoleParticipant UserRole = "participant"
)

// IsValid checks if the user role is valid
func (r UserRole) IsValid() bool {
	switch r {
	case RoleAdmin, RoleHost, RoleParticipant:
		return true
	}
	return false
}

// NewUser creates a new user with default values
func NewUser(email, name string) *User {
	now := time.Now()

	// Default notification preferences
	notifPrefs, _ := json.Marshal(map[string]interface{}{
		"email":   true,
		"push":    true,
		"reports": true,
	})

	// Default meeting preferences
	meetingPrefs, _ := json.Marshal(map[string]interface{}{
		"auto_join_audio": true,
		"auto_join_video": false,
	})

	return &User{
		ID:                      uuid.New(),
		Email:                   email,
		Name:                    name,
		Role:                    RoleParticipant,
		IsActive:                true,
		IsEmailVerified:         false,
		Timezone:                "UTC",
		Language:                "en",
		NotificationPreferences: notifPrefs,
		MeetingPreferences:      meetingPrefs,
		CreatedAt:               now,
		UpdatedAt:               now,
	}
}

// NewOAuthUser creates a new user from OAuth provider
func NewOAuthUser(email, name, provider, oauthID string) *User {
	user := NewUser(email, name)
	user.OAuthProvider = &provider
	user.OAuthID = &oauthID
	user.IsEmailVerified = true // OAuth providers verify emails
	return user
}

// UpdateLastLogin updates the last login timestamp
func (u *User) UpdateLastLogin() {
	now := time.Now()
	u.LastLoginAt = &now
	u.LastActiveAt = &now
	u.UpdatedAt = now
}

// UpdateActivity updates the last active timestamp
func (u *User) UpdateActivity() {
	now := time.Now()
	u.LastActiveAt = &now
	u.UpdatedAt = now
}

// CanHost checks if user can host meetings
func (u *User) CanHost() bool {
	return u.Role == RoleHost || u.Role == RoleAdmin
}

// IsAdmin checks if user is admin
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// Validate validates user data
func (u *User) Validate() error {
	if u.Email == "" {
		return ErrInvalidEmail
	}
	if u.Name == "" {
		return ErrInvalidName
	}
	if !u.Role.IsValid() {
		return ErrInvalidRole
	}
	return nil
}

// PublicUser returns a user with sensitive fields removed
type PublicUser struct {
	ID              uuid.UUID `json:"id"`
	Email           string    `json:"email"`
	Name            string    `json:"name"`
	Role            UserRole  `json:"role"`
	AvatarURL       *string   `json:"avatar_url,omitempty"`
	Bio             *string   `json:"bio,omitempty"`
	IsEmailVerified bool      `json:"is_email_verified"`
	CreatedAt       time.Time `json:"created_at"`
}

// ToPublic converts User to PublicUser
func (u *User) ToPublic() *PublicUser {
	return &PublicUser{
		ID:              u.ID,
		Email:           u.Email,
		Name:            u.Name,
		Role:            u.Role,
		AvatarURL:       u.AvatarURL,
		Bio:             u.Bio,
		IsEmailVerified: u.IsEmailVerified,
		CreatedAt:       u.CreatedAt,
	}
}
