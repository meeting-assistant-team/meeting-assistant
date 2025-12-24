package room

import "time"

// InviteByEmailRequest is the request to invite a user by email
type InviteByEmailRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// InviteByEmailResponse is the response after inviting a user
type InviteByEmailResponse struct {
	ParticipantID string `json:"participant_id"`
	Email         string `json:"email"`
	Status        string `json:"status"`
	Message       string `json:"message"`
}

// MyInvitationsResponse contains all invitations for the current user
type MyInvitationsResponse struct {
	Invitations []InvitationItem `json:"invitations"`
	Total       int              `json:"total"`
}

// InvitationItem represents a single invitation
type InvitationItem struct {
	ParticipantID string    `json:"participant_id"`
	RoomID        string    `json:"room_id"`
	RoomName      string    `json:"room_name"`
	RoomType      string    `json:"room_type"`
	InviterID     string    `json:"inviter_id"`
	InviterName   string    `json:"inviter_name"`
	InvitedAt     time.Time `json:"invited_at"`
}

// AcceptInvitationResponse is the response after accepting an invitation
type AcceptInvitationResponse struct {
	Status       string               `json:"status"`
	Message      string               `json:"message"`
	Room         *RoomResponse        `json:"room"`
	Participant  *ParticipantResponse `json:"participant"`
	LivekitToken string               `json:"livekit_token,omitempty"`
	LivekitURL   string               `json:"livekit_url,omitempty"`
}
