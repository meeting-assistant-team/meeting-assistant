package entities

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// WordTimestamp represents a single word with time and speaker info
type WordTimestamp struct {
	Word       string  `json:"word"`
	Start      float64 `json:"start"`
	End        float64 `json:"end"`
	Confidence float64 `json:"confidence"`
	Speaker    string  `json:"speaker,omitempty"`
}

// Segment represents a contiguous speech segment
type Segment struct {
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Text    string  `json:"text"`
	Speaker string  `json:"speaker"`
}

// Transcript is the stored transcript model
type Transcript struct {
	ID              uuid.UUID                                  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	MeetingID       uuid.UUID                                  `json:"meeting_id" gorm:"type:uuid;not null;index"`
	RecordingID     string                                     `json:"recording_id,omitempty" gorm:"type:varchar(255)"`
	RoomID          string                                     `json:"room_id,omitempty" gorm:"type:varchar(255);index"`
	Text            string                                     `json:"text" gorm:"type:text"`
	Summary         string                                     `json:"summary,omitempty" gorm:"type:text"`
	Chapters        []Chapter                                  `json:"chapters,omitempty" gorm:"type:jsonb;serializer:json"`
	Language        string                                     `json:"language,omitempty" gorm:"type:varchar(20)"`
	Segments        []Segment                                  `json:"segments,omitempty" gorm:"type:jsonb;serializer:json"`
	Words           []WordTimestamp                            `json:"words,omitempty" gorm:"type:jsonb;serializer:json"`
	ConfidenceScore float64                                    `json:"confidence_score,omitempty"`
	HasSpeakers     bool                                       `json:"has_speakers" gorm:"default:false"`
	SpeakerCount    int                                        `json:"speaker_count,omitempty"`
	ProcessingTime  int                                        `json:"processing_time,omitempty"` // in seconds
	ModelUsed       string                                     `json:"model_used,omitempty" gorm:"type:varchar(100)"`
	RawData         datatypes.JSONType[map[string]interface{}] `json:"raw_data,omitempty" gorm:"type:jsonb;serializer:json"`
	CreatedAt       time.Time                                  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time                                  `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for GORM
func (Transcript) TableName() string {
	return "transcripts"
}

// NewTranscript creates a new transcript
func NewTranscript(meetingID uuid.UUID) *Transcript {
	return &Transcript{
		ID:        uuid.New(),
		MeetingID: meetingID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
