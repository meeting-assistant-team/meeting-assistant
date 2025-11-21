package room

import (
	"time"

	"github.com/johnquangdev/meeting-assistant/internal/adapter/dto/auth"
)

// RoomResponse represents a room in responses
type RoomResponse struct {
	ID                  string                 `json:"id"`
	Name                string                 `json:"name"`
	Description         *string                `json:"description,omitempty"`
	Slug                *string                `json:"slug,omitempty"`
	HostID              string                 `json:"host_id"`
	Host                *auth.UserResponse     `json:"host,omitempty"`
	Type                string                 `json:"type"`
	Status              string                 `json:"status"`
	LivekitRoomName     string                 `json:"livekit_room_name"`
	MaxParticipants     int                    `json:"max_participants"`
	CurrentParticipants int                    `json:"current_participants"`
	Settings            map[string]interface{} `json:"settings"`
	ScheduledStartTime  *time.Time             `json:"scheduled_start_time,omitempty"`
	ScheduledEndTime    *time.Time             `json:"scheduled_end_time,omitempty"`
	StartedAt           *time.Time             `json:"started_at,omitempty"`
	EndedAt             *time.Time             `json:"ended_at,omitempty"`
	Duration            *int                   `json:"duration,omitempty"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

// CreateRoomResponse represents the response after creating a room
type CreateRoomResponse struct {
	Room         *RoomResponse `json:"room"`
	LivekitToken string        `json:"livekit_token"`
	LivekitURL   string        `json:"livekit_url"`
}

// ParticipantResponse represents a participant in responses
type ParticipantResponse struct {
	ID                string             `json:"id"`
	RoomID            string             `json:"room_id"`
	UserID            string             `json:"user_id"`
	User              *auth.UserResponse `json:"user,omitempty"`
	Role              string             `json:"role"`
	Status            string             `json:"status"`
	JoinedAt          *time.Time         `json:"joined_at,omitempty"`
	LeftAt            *time.Time         `json:"left_at,omitempty"`
	Duration          *int               `json:"duration,omitempty"`
	CanShareScreen    bool               `json:"can_share_screen"`
	CanRecord         bool               `json:"can_record"`
	CanMuteOthers     bool               `json:"can_mute_others"`
	IsMuted           bool               `json:"is_muted"`
	IsHandRaised      bool               `json:"is_hand_raised"`
	ConnectionQuality *string            `json:"connection_quality,omitempty"`
	CreatedAt         time.Time          `json:"created_at"`
}

// JoinRoomResponse represents the response after joining a room
type JoinRoomResponse struct {
	Status       string               `json:"status"`                  // "joined" or "waiting"
	Message      string               `json:"message"`                 // User-friendly message
	Room         *RoomResponse        `json:"room"`                    // Room information
	Participant  *ParticipantResponse `json:"participant"`             // Current user's participant record
	LivekitToken string               `json:"livekit_token,omitempty"` // Only for joined status
	LivekitURL   string               `json:"livekit_url,omitempty"`   // Only for joined status
}

// RoomListResponse represents a paginated list of rooms
type RoomListResponse struct {
	Rooms      []*RoomResponse `json:"rooms"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}

// ParticipantListResponse represents a list of participants
type ParticipantListResponse struct {
	Participants []*ParticipantResponse `json:"participants"`
	Total        int                    `json:"total"`
}
