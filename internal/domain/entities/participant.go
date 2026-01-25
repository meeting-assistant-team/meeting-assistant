package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// ParticipantRole represents the role of a participant in a room
type ParticipantRole string

const (
	ParticipantRoleHost        ParticipantRole = "host"
	ParticipantRoleCoHost      ParticipantRole = "co_host"
	ParticipantRoleParticipant ParticipantRole = "participant"
	ParticipantRoleGuest       ParticipantRole = "guest"
)

// ParticipantStatus represents the status of a participant in their room lifecycle
type ParticipantStatus string

const (
	// ParticipantStatusInvited: User has been invited to the room but hasn't responded yet
	// Used when: Host sends invitation via email or in-app notification
	ParticipantStatusInvited ParticipantStatus = "invited"

	// ParticipantStatusWaiting: User is in the waiting room, pending host approval
	// Used when: Room has waiting room enabled and user is waiting for host to admit them
	ParticipantStatusWaiting ParticipantStatus = "waiting"

	// ParticipantStatusJoined: User is actively in the room (connected to LiveKit)
	// Used when: User successfully joined the meeting and is currently present
	ParticipantStatusJoined ParticipantStatus = "joined"

	// ParticipantStatusLeft: User has left the room normally (voluntary exit)
	// Used when: User clicks "Leave" button or closes the meeting window
	ParticipantStatusLeft ParticipantStatus = "left"

	// ParticipantStatusRemoved: User was forcibly removed by host/co-host
	// Used when: Host kicks out a participant (includes removed_by and removal_reason)
	ParticipantStatusRemoved ParticipantStatus = "removed"

	// ParticipantStatusDeclined: User explicitly declined the invitation
	// Used when: User clicks "Decline" on invitation notification/email
	ParticipantStatusDeclined ParticipantStatus = "declined"

	// ParticipantStatusDenied: Host denied user's request to join (waiting room rejection)
	// Reserved for future "block" feature - currently unused (deny = delete participant record)
	// Used when: Host clicks "Deny" on waiting room request
	ParticipantStatusDenied ParticipantStatus = "denied"
)

// Participant represents a user's participation in a room
type Participant struct {
	ID     uuid.UUID         `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	RoomID uuid.UUID         `gorm:"type:uuid;not null;index" json:"room_id"`
	Room   *Room             `gorm:"foreignKey:RoomID" json:"room,omitempty"`
	UserID *uuid.UUID        `gorm:"type:uuid;index" json:"user_id,omitempty"`
	User   *User             `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Role   ParticipantRole   `gorm:"type:varchar(20);default:'participant'" json:"role"`
	Status ParticipantStatus `gorm:"type:varchar(20);default:'invited';index" json:"status"`

	// Invitation fields
	InvitedEmail      *string        `gorm:"type:varchar(255);index" json:"invited_email,omitempty"`
	InvitedBy         *uuid.UUID     `gorm:"type:uuid;index" json:"invited_by,omitempty"`
	InvitedAt         *time.Time     `json:"invited_at,omitempty"`
	JoinedAt          *time.Time     `gorm:"index" json:"joined_at,omitempty"`
	LeftAt            *time.Time     `json:"left_at,omitempty"`
	Duration          *int           `json:"duration,omitempty"` // seconds in meeting
	CanShareScreen    bool           `gorm:"default:true" json:"can_share_screen"`
	CanRecord         bool           `gorm:"default:false" json:"can_record"`
	CanMuteOthers     bool           `gorm:"default:false" json:"can_mute_others"`
	IsMuted           bool           `gorm:"default:false" json:"is_muted"`
	IsHandRaised      bool           `gorm:"default:false" json:"is_hand_raised"`
	IsRemoved         bool           `gorm:"default:false" json:"is_removed"`
	RemovedBy         *uuid.UUID     `gorm:"type:uuid" json:"removed_by,omitempty"`
	RemovalReason     *string        `gorm:"type:text" json:"removal_reason,omitempty"`
	ConnectionQuality *string        `gorm:"type:varchar(20)" json:"connection_quality,omitempty"` // excellent, good, poor
	DeviceInfo        datatypes.JSON `gorm:"type:jsonb" json:"device_info,omitempty"`
	Metadata          datatypes.JSON `gorm:"type:jsonb;default:'{}'" json:"metadata,omitempty"`
	CreatedAt         time.Time      `gorm:"default:now()" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"default:now()" json:"updated_at"`
}

// TableName specifies the table name for Participant
func (Participant) TableName() string {
	return "participants"
}

// IsHost checks if the participant is a host
func (p *Participant) IsHost() bool {
	return p.Role == ParticipantRoleHost || p.Role == ParticipantRoleCoHost
}

// IsActive checks if the participant is currently in the room
func (p *Participant) IsActive() bool {
	return p.Status == ParticipantStatusJoined && p.LeftAt == nil
}

// Join marks the participant as joined
func (p *Participant) Join() {
	now := time.Now()
	p.Status = ParticipantStatusJoined
	p.JoinedAt = &now
}

// Leave marks the participant as left and calculates duration
func (p *Participant) Leave() {
	now := time.Now()
	p.Status = ParticipantStatusLeft
	p.LeftAt = &now

	if p.JoinedAt != nil {
		duration := int(now.Sub(*p.JoinedAt).Seconds())
		p.Duration = &duration
	}
}

// Remove marks the participant as removed
func (p *Participant) Remove(removedBy uuid.UUID, reason string) {
	now := time.Now()
	p.Status = ParticipantStatusRemoved
	p.IsRemoved = true
	p.RemovedBy = &removedBy
	p.RemovalReason = &reason
	p.LeftAt = &now

	if p.JoinedAt != nil {
		duration := int(now.Sub(*p.JoinedAt).Seconds())
		p.Duration = &duration
	}
}

// PromoteToHost promotes the participant to host role
func (p *Participant) PromoteToHost() {
	p.Role = ParticipantRoleHost
	p.CanRecord = true
	p.CanMuteOthers = true
}

// PromoteToCoHost promotes the participant to co-host role
func (p *Participant) PromoteToCoHost() {
	p.Role = ParticipantRoleCoHost
	p.CanRecord = true
	p.CanMuteOthers = true
}
