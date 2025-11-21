package room

import (
	"time"
)

// CreateRoomRequest represents the request to create a room
type CreateRoomRequest struct {
	Name               string                 `json:"name" validate:"required,min=1,max=255"`
	Description        *string                `json:"description,omitempty"`
	Type               string                 `json:"type" validate:"required,oneof=public private scheduled"`
	MaxParticipants    int                    `json:"max_participants" validate:"required,min=2,max=100"`
	Settings           map[string]interface{} `json:"settings,omitempty"`
	ScheduledStartTime *time.Time             `json:"scheduled_start_time,omitempty"`
	ScheduledEndTime   *time.Time             `json:"scheduled_end_time,omitempty"`
}

// UpdateRoomRequest represents the request to update a room
type UpdateRoomRequest struct {
	Name               *string                `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description        *string                `json:"description,omitempty"`
	MaxParticipants    *int                   `json:"max_participants,omitempty" validate:"omitempty,min=2,max=100"`
	Settings           map[string]interface{} `json:"settings,omitempty"`
	ScheduledStartTime *time.Time             `json:"scheduled_start_time,omitempty"`
	ScheduledEndTime   *time.Time             `json:"scheduled_end_time,omitempty"`
}

// ListRoomsRequest represents query parameters for listing rooms
type ListRoomsRequest struct {
	Type      *string  `query:"type" validate:"omitempty,oneof=public private scheduled"`
	Status    *string  `query:"status" validate:"omitempty,oneof=scheduled active ended cancelled"`
	Search    string   `query:"search"`
	Tags      []string `query:"tags"`
	Page      int      `query:"page" validate:"min=1"`
	PageSize  int      `query:"page_size" validate:"min=1,max=100"`
	SortBy    string   `query:"sort_by" validate:"omitempty,oneof=created_at started_at name"`
	SortOrder string   `query:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// JoinRoomRequest represents the request to join a room
type JoinRoomRequest struct {
	Token *string `json:"token,omitempty"` // For private rooms
}

// RemoveParticipantRequest represents the request to remove a participant
type RemoveParticipantRequest struct {
	Reason string `json:"reason" validate:"required,min=1,max=500"`
}

// TransferHostRequest represents the request to transfer host role
type TransferHostRequest struct {
	NewHostID string `json:"new_host_id" validate:"required,uuid"`
}

// UpdateParticipantRequest represents the request to update participant settings
type UpdateParticipantRequest struct {
	IsMuted      *bool `json:"is_muted,omitempty"`
	IsHandRaised *bool `json:"is_hand_raised,omitempty"`
}

// DenyParticipantRequest represents the request to deny a participant
type DenyParticipantRequest struct {
	Reason string `json:"reason,omitempty"`
}
