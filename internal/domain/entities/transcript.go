package entities

import "time"

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
	ID              string          `json:"id"`
	RecordingID     string          `json:"recording_id"`
	RoomID          string          `json:"room_id"`
	Text            string          `json:"text"`
	Language        string          `json:"language"`
	Segments        []Segment       `json:"segments"`
	Words           []WordTimestamp `json:"words"`
	ConfidenceScore float64         `json:"confidence_score"`
	HasSpeakers     bool            `json:"has_speakers"`
	SpeakerCount    int             `json:"speaker_count"`
	ProcessingTime  int             `json:"processing_time"`
	ModelUsed       string          `json:"model_used"`
	CreatedAt       time.Time       `json:"created_at"`
}
