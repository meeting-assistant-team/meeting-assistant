package entities

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

// TokenFamily represents a refresh token rotation family for security tracking
type TokenFamily struct {
	ID               uuid.UUID              `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID           uuid.UUID              `json:"user_id" gorm:"type:uuid;not null;index"`
	FamilyID         uuid.UUID              `json:"family_id" gorm:"type:uuid;not null;index"`
	RefreshTokenHash string                 `json:"-" gorm:"column:refresh_token_hash;type:varchar(255);uniqueIndex;not null"`
	ParentTokenHash  *string                `json:"-" gorm:"column:parent_token_hash;type:varchar(255)"`
	CreatedAt        time.Time              `json:"created_at" gorm:"autoCreateTime"`
	ExpiresAt        time.Time              `json:"expires_at" gorm:"type:timestamp;not null;index"`
	RevokedAt        *time.Time             `json:"revoked_at,omitempty" gorm:"type:timestamp"`
	LastUsedAt       *time.Time             `json:"last_used_at,omitempty" gorm:"type:timestamp"`
	DeviceInfo       map[string]interface{} `json:"device_info,omitempty" gorm:"type:jsonb"`
	IPAddress        *string                `json:"ip_address,omitempty" gorm:"type:inet"`
	UserAgent        *string                `json:"user_agent,omitempty" gorm:"type:text"`
}

// TableName specifies the table name for GORM
func (TokenFamily) TableName() string {
	return "token_families"
}

// HashToken creates SHA256 hash of a refresh token
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// NewTokenFamily creates a new token family (initial token in chain)
func NewTokenFamily(userID uuid.UUID, refreshToken string, expiresAt time.Time) *TokenFamily {
	return &TokenFamily{
		ID:               uuid.New(),
		UserID:           userID,
		FamilyID:         uuid.New(), // New family for new login
		RefreshTokenHash: HashToken(refreshToken),
		ParentTokenHash:  nil, // No parent - this is the root
		ExpiresAt:        expiresAt,
		CreatedAt:        time.Now(),
	}
}

// RotateToken creates a new token family entry for token rotation
func (tf *TokenFamily) RotateToken(newRefreshToken string, expiresAt time.Time) *TokenFamily {
	oldTokenHash := tf.RefreshTokenHash
	return &TokenFamily{
		ID:               uuid.New(),
		UserID:           tf.UserID,
		FamilyID:         tf.FamilyID, // Same family ID - part of rotation chain
		RefreshTokenHash: HashToken(newRefreshToken),
		ParentTokenHash:  &oldTokenHash, // Track previous token
		ExpiresAt:        expiresAt,
		CreatedAt:        time.Now(),
	}
}

// IsExpired checks if token family is expired
func (tf *TokenFamily) IsExpired() bool {
	return time.Now().After(tf.ExpiresAt)
}

// IsValid checks if token family is valid (not expired and not revoked)
func (tf *TokenFamily) IsValid() bool {
	if tf == nil {
		return false
	}
	return !tf.IsExpired() && tf.RevokedAt == nil
}

// Revoke revokes the token family
func (tf *TokenFamily) Revoke() {
	now := time.Now()
	tf.RevokedAt = &now
}

// UpdateLastUsed updates the last used timestamp
func (tf *TokenFamily) UpdateLastUsed() {
	now := time.Now()
	tf.LastUsedAt = &now
}

// WithDeviceInfo adds device information
func (tf *TokenFamily) WithDeviceInfo(ip, userAgent string) *TokenFamily {
	tf.IPAddress = &ip
	tf.UserAgent = &userAgent
	return tf
}
