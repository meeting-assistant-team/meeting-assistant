package entities

import (
	"time"

	"github.com/google/uuid"
)

// TranscriptUtterance represents a single speaker segment/turn in a conversation
type TranscriptUtterance struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TranscriptID uuid.UUID `json:"transcript_id" gorm:"type:uuid;not null;index"`
	Speaker      string    `json:"speaker" gorm:"type:varchar(50);not null"`
	Text         string    `json:"text" gorm:"type:text;not null"`
	StartTime    float64   `json:"start_time" gorm:"not null"`
	EndTime      float64   `json:"end_time" gorm:"not null"`
	Confidence   float64   `json:"confidence" gorm:"default:0.0"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for GORM
func (TranscriptUtterance) TableName() string {
	return "transcript_utterances"
}

// Chapter represents an auto-generated chapter in a transcript
type Chapter struct {
	Gist     string  `json:"gist"`
	Headline string  `json:"headline"`
	Summary  string  `json:"summary"`
	Start    float64 `json:"start"`
	End      float64 `json:"end"`
}
