package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// RecordingStatus represents the status of a recording
type RecordingStatus string

const (
	RecordingStatusRecording  RecordingStatus = "recording"
	RecordingStatusProcessing RecordingStatus = "processing"
	RecordingStatusCompleted  RecordingStatus = "completed"
	RecordingStatusFailed     RecordingStatus = "failed"
	RecordingStatusDeleted    RecordingStatus = "deleted"
)

// Recording represents a meeting recording
type Recording struct {
	ID                    uuid.UUID       `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	RoomID                uuid.UUID       `json:"room_id" gorm:"type:uuid;not null;index"`
	StartedBy             *uuid.UUID      `json:"started_by,omitempty" gorm:"type:uuid"`
	LivekitRecordingID    *string         `json:"livekit_recording_id,omitempty" gorm:"type:varchar(255);unique"`
	LivekitEgressID       *string         `json:"livekit_egress_id,omitempty" gorm:"type:varchar(255)"`
	Status                RecordingStatus `json:"status" gorm:"type:varchar(20);not null;default:'recording';index"`
	FileURL               *string         `json:"file_url,omitempty" gorm:"type:text"`
	FilePath              *string         `json:"file_path,omitempty" gorm:"type:text"`
	FileSize              *int64          `json:"file_size,omitempty"`
	FileFormat            string          `json:"file_format" gorm:"type:varchar(20);default:'mp4'"`
	Duration              *int            `json:"duration,omitempty"`
	StartedAt             time.Time       `json:"started_at" gorm:"not null;default:now()"`
	CompletedAt           *time.Time      `json:"completed_at,omitempty"`
	ProcessingStartedAt   *time.Time      `json:"processing_started_at,omitempty"`
	ProcessingCompletedAt *time.Time      `json:"processing_completed_at,omitempty"`
	ProcessingError       *string         `json:"processing_error,omitempty" gorm:"type:text"`
	VideoTracks           int             `json:"video_tracks" gorm:"default:0"`
	AudioTracks           int             `json:"audio_tracks" gorm:"default:0"`
	Resolution            *string         `json:"resolution,omitempty" gorm:"type:varchar(20)"`
	Bitrate               *int            `json:"bitrate,omitempty"`
	Metadata              datatypes.JSON  `json:"metadata,omitempty" gorm:"type:jsonb;default:'{}'"`
	CreatedAt             time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt             time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for GORM
func (Recording) TableName() string {
	return "recordings"
}

// IsCompleted checks if recording is completed
func (r *Recording) IsCompleted() bool {
	return r.Status == RecordingStatusCompleted
}

// IsFailed checks if recording failed
func (r *Recording) IsFailed() bool {
	return r.Status == RecordingStatusFailed
}

// MarkAsProcessing marks recording as processing
func (r *Recording) MarkAsProcessing() {
	r.Status = RecordingStatusProcessing
	now := time.Now()
	r.ProcessingStartedAt = &now
}

// MarkAsCompleted marks recording as completed
func (r *Recording) MarkAsCompleted() {
	r.Status = RecordingStatusCompleted
	now := time.Now()
	r.CompletedAt = &now
	r.ProcessingCompletedAt = &now
}

// MarkAsFailed marks recording as failed
func (r *Recording) MarkAsFailed(errorMsg string) {
	r.Status = RecordingStatusFailed
	r.ProcessingError = &errorMsg
	now := time.Now()
	r.ProcessingCompletedAt = &now
}
