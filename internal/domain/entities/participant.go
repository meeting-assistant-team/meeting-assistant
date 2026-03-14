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
	// ParticipantStatusInvited: Người dùng đã được mời vào phòng nhưng chưa phản hồi
	// Sử dụng khi: Host gửi lời mời qua email hoặc thông báo trong ứng dụng
	ParticipantStatusInvited ParticipantStatus = "invited"

	// ParticipantStatusWaiting: Người dùng đang ở phòng chờ, chờ host chấp nhận
	// Sử dụng khi: Phòng có bật chế độ phòng chờ và người dùng đang chờ host cho phép vào
	ParticipantStatusWaiting ParticipantStatus = "waiting"

	// ParticipantStatusJoined: Người dùng đang tham gia phòng (đã kết nối với LiveKit)
	// Sử dụng khi: Người dùng đã vào meeting thành công và hiện đang có mặt
	ParticipantStatusJoined ParticipantStatus = "joined"

	// ParticipantStatusLeft: Người dùng đã rời phòng một cách bình thường (tự nguyện)
	// Sử dụng khi: Người dùng nhấn nút 'Leave' hoặc đóng cửa sổ meeting
	ParticipantStatusLeft ParticipantStatus = "left"

	// ParticipantStatusRemoved: Người dùng bị host/co-host kick ra khỏi phòng
	// Sử dụng khi: Host kick một người tham gia (bao gồm removed_by và removal_reason)
	ParticipantStatusRemoved ParticipantStatus = "removed"

	// ParticipantStatusDeclined: Người dùng từ chối lời mời một cách rõ ràng
	// Sử dụng khi: Người dùng nhấn 'Decline' trên thông báo/email mời
	ParticipantStatusDeclined ParticipantStatus = "declined"

	// ParticipantStatusDenied: Host từ chối yêu cầu tham gia của người dùng (từ chối từ phòng chờ)
	// Dành riêng cho tính năng 'block' trong tương lai - hiện chưa sử dụng (deny = xóa bản ghi participant)
	// Sử dụng khi: Host nhấn 'Deny' trên yêu cầu từ phòng chờ
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
