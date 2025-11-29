package entities

import "time"

type MeetingSummary struct {
	ID                 string              `json:"id"`
	RoomID             string              `json:"room_id"`
	TranscriptID       string              `json:"transcript_id"`
	ExecutiveSummary   string              `json:"executive_summary"`
	KeyPoints          []string            `json:"key_points"`
	Decisions          []map[string]string `json:"decisions"`
	Topics             []string            `json:"topics"`
	OpenQuestions      []string            `json:"open_questions"`
	NextSteps          []string            `json:"next_steps"`
	OverallSentiment   float64             `json:"overall_sentiment"`
	SentimentBreakdown map[string]float64  `json:"sentiment_breakdown"`
	ModelUsed          string              `json:"model_used"`
	ProcessingTime     int                 `json:"processing_time"`
	CreatedAt          time.Time           `json:"created_at"`
}
