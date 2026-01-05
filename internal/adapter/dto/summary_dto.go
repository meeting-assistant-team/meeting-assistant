package dto

import (
	"time"

	"github.com/google/uuid"
)

// MeetingSummaryResponse represents the API response for meeting summary
type MeetingSummaryResponse struct {
	ID                 uuid.UUID              `json:"id"`
	RoomID             uuid.UUID              `json:"room_id"`
	TranscriptID       uuid.UUID              `json:"transcript_id"`
	ExecutiveSummary   string                 `json:"executive_summary"`
	KeyPoints          []KeyPoint             `json:"key_points"`
	Decisions          []Decision             `json:"decisions"`
	Topics             []string               `json:"topics"`
	KeyQuestions       []string               `json:"key_questions"`
	Chapters           []Chapter              `json:"chapters,omitempty"`
	ActionItems        []ActionItemDTO        `json:"action_items"`
	SentimentBreakdown map[string]interface{} `json:"sentiment_breakdown"`
	EngagementMetrics  EngagementMetricsDTO   `json:"engagement_metrics"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

// KeyPoint represents a key point from the meeting
type KeyPoint struct {
	Text             string `json:"text"`
	TimestampSeconds int    `json:"timestamp_seconds"`
	MentionedBy      string `json:"mentioned_by"`
	Importance       string `json:"importance"` // high, medium, low
}

// Decision represents a decision made in the meeting
type Decision struct {
	DecisionText     string `json:"decision_text"`
	Owner            string `json:"owner"`
	TimestampSeconds int    `json:"timestamp_seconds"`
	Impact           string `json:"impact"` // high, medium, low
}

// NextStep represents a next step or follow-up
type NextStep struct {
	Description      string `json:"description"`
	Owner            string `json:"owner"`
	DueDateMentioned string `json:"due_date_mentioned,omitempty"`
	Priority         string `json:"priority"` // high, medium, low
}

// Chapter represents a meeting chapter/section
type Chapter struct {
	Gist     string `json:"gist"`
	Headline string `json:"headline"`
	Summary  string `json:"summary"`
	Start    float64 `json:"start"` // Start time in milliseconds
	End      float64 `json:"end"`   // End time in milliseconds
}

// ActionItemDTO represents an action item
type ActionItemDTO struct {
	ID                  uuid.UUID  `json:"id"`
	Title               string     `json:"title"`
	Description         string     `json:"description,omitempty"`
	AssignedTo          *string    `json:"assigned_to,omitempty"`
	Type                string     `json:"type"`     // action, decision, question, follow_up
	Priority            string     `json:"priority"` // low, medium, high, urgent
	Status              string     `json:"status"`   // pending, in_progress, completed, cancelled
	DueDate             *time.Time `json:"due_date,omitempty"`
	TranscriptReference string     `json:"transcript_reference,omitempty"`
	TimestampInMeeting  int        `json:"timestamp_in_meeting,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

// EngagementMetricsDTO represents engagement metrics
type EngagementMetricsDTO struct {
	TotalSpeakingTime       int     `json:"total_speaking_time_seconds"`
	EngagementScore         float64 `json:"engagement_score"`          // 0-1
	ParticipantBalanceScore float64 `json:"participant_balance_score"` // 0-1
}

// ParticipantMetric represents metrics for a single participant
type ParticipantMetric struct {
	SpeakingTimeSeconds int     `json:"speaking_time_seconds"`
	SpeakingPercentage  float64 `json:"speaking_percentage"`
	TurnCount           int     `json:"turn_count"`
	Sentiment           float64 `json:"sentiment"`
	EngagementLevel     string  `json:"engagement_level"` // high, medium, low
}

// ProcessingInfo represents processing information
type ProcessingInfo struct {
	ModelUsed      string `json:"model_used"`
	ProcessingTime int    `json:"processing_time_ms"`
	Language       string `json:"language,omitempty"`
}

// SummaryStatusResponse represents the status of summary generation
type SummaryStatusResponse struct {
	Status      string    `json:"status"` // pending, transcript_ready, summarizing, completed, failed
	Message     string    `json:"message"`
	RoomID      uuid.UUID `json:"room_id"`
	JobID       uuid.UUID `json:"job_id"`
	SubmittedAt time.Time `json:"submitted_at"`
}

// CreateActionItemRequest represents the request to create an action item
type CreateActionItemRequest struct {
	Title               string     `json:"title" validate:"required,min=3,max=500"`
	Description         string     `json:"description,omitempty"`
	AssignedTo          *uuid.UUID `json:"assigned_to,omitempty"`
	Type                string     `json:"type" validate:"required,oneof=action decision question follow_up research"`
	Priority            string     `json:"priority" validate:"required,oneof=low medium high urgent"`
	DueDate             *time.Time `json:"due_date,omitempty"`
	EstimatedHours      float64    `json:"estimated_hours,omitempty"`
	TranscriptReference string     `json:"transcript_reference,omitempty"`
	TimestampInMeeting  int        `json:"timestamp_in_meeting,omitempty"`
}

// UpdateActionItemRequest represents the request to update an action item
type UpdateActionItemRequest struct {
	Title          *string    `json:"title,omitempty" validate:"omitempty,min=3,max=500"`
	Description    *string    `json:"description,omitempty"`
	AssignedTo     *uuid.UUID `json:"assigned_to,omitempty"`
	Priority       *string    `json:"priority,omitempty" validate:"omitempty,oneof=low medium high urgent"`
	Status         *string    `json:"status,omitempty" validate:"omitempty,oneof=pending in_progress completed cancelled blocked"`
	DueDate        *time.Time `json:"due_date,omitempty"`
	EstimatedHours *float64   `json:"estimated_hours,omitempty"`
}

// ListActionItemsResponse represents the response for listing action items
type ListActionItemsResponse struct {
	ActionItems []ActionItemDTO     `json:"action_items"`
	Pagination  *PaginationResponse `json:"pagination,omitempty"`
}

// PaginationResponse represents pagination metadata
type PaginationResponse struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalItems int `json:"total_items"`
	TotalPages int `json:"total_pages"`
}
