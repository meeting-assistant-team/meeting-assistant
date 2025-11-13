package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// RoomType represents the type of room
type RoomType string

const (
	RoomTypePublic    RoomType = "public"
	RoomTypePrivate   RoomType = "private"
	RoomTypeScheduled RoomType = "scheduled"
)

// RoomStatus represents the current status of a room
type RoomStatus string

const (
	RoomStatusScheduled RoomStatus = "scheduled"
	RoomStatusActive    RoomStatus = "active"
	RoomStatusEnded     RoomStatus = "ended"
	RoomStatusCancelled RoomStatus = "cancelled"
)

// Room represents a meeting room
type Room struct {
	ID                  uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name                string         `gorm:"type:varchar(255);not null" json:"name"`
	Description         *string        `gorm:"type:text" json:"description,omitempty"`
	Slug                *string        `gorm:"type:varchar(100);unique" json:"slug,omitempty"`
	HostID              uuid.UUID      `gorm:"type:uuid;not null;index" json:"host_id"`
	Host                *User          `gorm:"foreignKey:HostID" json:"host,omitempty"`
	Type                RoomType       `gorm:"type:varchar(20);not null;default:'public';index" json:"type"`
	Status              RoomStatus     `gorm:"type:varchar(20);not null;default:'scheduled';index" json:"status"`
	LivekitRoomName     string         `gorm:"type:varchar(255);unique;not null" json:"livekit_room_name"`
	LivekitRoomID       *string        `gorm:"type:varchar(255)" json:"livekit_room_id,omitempty"`
	MaxParticipants     int            `gorm:"default:10;check:max_participants >= 2 AND max_participants <= 100" json:"max_participants"`
	CurrentParticipants int            `gorm:"default:0" json:"current_participants"`
	Settings            datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"settings"`
	ScheduledStartTime  *time.Time     `gorm:"index" json:"scheduled_start_time,omitempty"`
	ScheduledEndTime    *time.Time     `json:"scheduled_end_time,omitempty"`
	StartedAt           *time.Time     `json:"started_at,omitempty"`
	EndedAt             *time.Time     `json:"ended_at,omitempty"`
	Duration            *int           `json:"duration,omitempty"` // seconds
	Tags                datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"tags,omitempty"`
	Metadata            datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"metadata,omitempty"`
	CreatedAt           time.Time      `gorm:"default:now()" json:"created_at"`
	UpdatedAt           time.Time      `gorm:"default:now()" json:"updated_at"`
}

// TableName specifies the table name for Room
func (Room) TableName() string {
	return "rooms"
}

// DefaultSettings returns default room settings
func DefaultSettings() map[string]interface{} {
	return map[string]interface{}{
		"enable_recording":      true,
		"enable_chat":           true,
		"enable_screen_share":   true,
		"require_approval":      false,
		"allow_guests":          false,
		"mute_on_join":          false,
		"disable_video_on_join": false,
		"enable_waiting_room":   false,
		"auto_record":           false,
		"enable_transcription":  true,
	}
}

// IsActive checks if the room is currently active
func (r *Room) IsActive() bool {
	return r.Status == RoomStatusActive
}

// IsEnded checks if the room has ended
func (r *Room) IsEnded() bool {
	return r.Status == RoomStatusEnded
}

// IsFull checks if the room has reached max capacity
func (r *Room) IsFull() bool {
	return r.CurrentParticipants >= r.MaxParticipants
}

// CanJoin checks if a user can join this room
func (r *Room) CanJoin() bool {
	return r.IsActive() && !r.IsFull()
}

// Start marks the room as active
func (r *Room) Start() {
	now := time.Now()
	r.Status = RoomStatusActive
	r.StartedAt = &now
}

// End marks the room as ended and calculates duration
func (r *Room) End() {
	now := time.Now()
	r.Status = RoomStatusEnded
	r.EndedAt = &now

	if r.StartedAt != nil {
		duration := int(now.Sub(*r.StartedAt).Seconds())
		r.Duration = &duration
	}
}

// IncrementParticipants increases the participant count
func (r *Room) IncrementParticipants() {
	r.CurrentParticipants++
}

// DecrementParticipants decreases the participant count
func (r *Room) DecrementParticipants() {
	if r.CurrentParticipants > 0 {
		r.CurrentParticipants--
	}
}
