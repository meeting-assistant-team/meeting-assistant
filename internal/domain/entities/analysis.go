package entities

import (
	"time"

	"github.com/google/uuid"
)

// AnalysisResult represents the structured output from Groq LLM analysis
type AnalysisResult struct {
	ExecutiveSummary   string                        `json:"executive_summary"`
	KeyPoints          []KeyPoint                    `json:"key_points"`
	Decisions          []Decision                    `json:"decisions"`
	Topics             []string                      `json:"topics"`
	KeyQuestions       []string                      `json:"key_questions"`
	NextSteps          []NextStep                    `json:"next_steps"`
	ActionItems        []ActionItemExtracted         `json:"action_items"`
	OverallSentiment   float64                       `json:"overall_sentiment"`
	SpeakerSentiment   map[string]float64            `json:"speaker_sentiment"`
	EngagementScore    float64                       `json:"engagement_score"`
	ParticipantBalance map[string]ParticipantMetrics `json:"participant_balance"`
}

// KeyPoint represents a key point discussed in the meeting
type KeyPoint struct {
	Text               string `json:"text"`
	TimestampSeconds   int    `json:"timestamp_seconds"`
	MentionedBySpeaker string `json:"mentioned_by_speaker"`
	Importance         string `json:"importance"` // low, medium, high
}

// Decision represents a decision made during the meeting
type Decision struct {
	DecisionText     string `json:"decision_text"`
	Owner            string `json:"owner"`
	TimestampSeconds int    `json:"timestamp_seconds"`
	Impact           string `json:"impact"` // low, medium, high
}

// NextStep represents a next step or follow-up action
type NextStep struct {
	Description      string `json:"description"`
	Owner            string `json:"owner"`
	DueDateMentioned string `json:"due_date_mentioned"` // e.g., "next week", "by Friday"
	Priority         string `json:"priority"`           // low, medium, high, urgent
}

// ActionItemExtracted represents an action item extracted from transcript
type ActionItemExtracted struct {
	Title               string `json:"title"`
	Description         string `json:"description"`
	AssignedTo          string `json:"assigned_to"`
	Type                string `json:"type"`     // action, decision, question, follow_up
	Priority            string `json:"priority"` // low, medium, high, urgent
	TranscriptReference string `json:"transcript_reference"`
	TimestampInMeeting  int    `json:"timestamp_in_meeting"`
}

// ParticipantMetrics represents engagement metrics for a participant
type ParticipantMetrics struct {
	SpeakingTimeSeconds int     `json:"speaking_time_seconds"`
	SpeakingPercentage  float64 `json:"speaking_percentage"`
	TurnCount           int     `json:"turn_count"`
	Sentiment           float64 `json:"sentiment"`
	EngagementLevel     string  `json:"engagement_level"` // low, medium, high
}

// MeetingSummary represents the complete analysis of a meeting
type MeetingSummary struct {
	ID                 uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	RoomID             uuid.UUID `json:"room_id" gorm:"type:uuid;not null;uniqueIndex"`
	TranscriptID       uuid.UUID `json:"transcript_id" gorm:"type:uuid;index"`
	ExecutiveSummary   string    `json:"executive_summary" gorm:"type:text;not null"`
	KeyPoints          []byte    `json:"key_points,omitempty" gorm:"type:jsonb"`
	Decisions          []byte    `json:"decisions,omitempty" gorm:"type:jsonb"`
	Topics             []byte    `json:"topics,omitempty" gorm:"type:jsonb"`
	OpenQuestions      []byte    `json:"open_questions,omitempty" gorm:"type:jsonb"`
	NextSteps          []byte    `json:"next_steps,omitempty" gorm:"type:jsonb"`
	OverallSentiment   float64   `json:"overall_sentiment,omitempty"`
	SentimentBreakdown []byte    `json:"sentiment_breakdown,omitempty" gorm:"type:jsonb"`
	TotalSpeakingTime  int       `json:"total_speaking_time,omitempty"`
	ParticipantBalance float64   `json:"participant_balance_score,omitempty"`
	EngagementScore    float64   `json:"engagement_score,omitempty"`
	ModelUsed          string    `json:"model_used,omitempty" gorm:"type:varchar(50)"`
	ProcessingTime     int       `json:"processing_time,omitempty"` // in milliseconds
	Metadata           []byte    `json:"metadata,omitempty" gorm:"type:jsonb"`
	CreatedAt          time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt          time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for MeetingSummary
func (MeetingSummary) TableName() string {
	return "meeting_summaries"
}

// NewMeetingSummary creates a new MeetingSummary entity
func NewMeetingSummary(roomID, transcriptID uuid.UUID) *MeetingSummary {
	return &MeetingSummary{
		ID:           uuid.New(),
		RoomID:       roomID,
		TranscriptID: transcriptID,
		ModelUsed:    "groq",
	}
}

// ActionItem represents a task or action item from the meeting
type ActionItem struct {
	ID                  uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	RoomID              uuid.UUID  `json:"room_id" gorm:"type:uuid;not null;index"`
	SummaryID           *uuid.UUID `json:"summary_id,omitempty" gorm:"type:uuid;index"`
	AssignedTo          *uuid.UUID `json:"assigned_to,omitempty" gorm:"type:uuid"`
	CreatedBy           *uuid.UUID `json:"created_by,omitempty" gorm:"type:uuid"`
	Title               string     `json:"title" gorm:"type:varchar(500);not null"`
	Description         string     `json:"description,omitempty" gorm:"type:text"`
	Type                string     `json:"type" gorm:"type:varchar(50);default:'action'"`
	Priority            string     `json:"priority" gorm:"type:varchar(20);default:'medium'"`
	Status              string     `json:"status" gorm:"type:varchar(20);default:'pending'"`
	DueDate             *time.Time `json:"due_date,omitempty"`
	EstimatedHours      float64    `json:"estimated_hours,omitempty"`
	TranscriptReference string     `json:"transcript_reference,omitempty" gorm:"type:text"`
	TimestampInMeeting  int        `json:"timestamp_in_meeting,omitempty"`
	ClickupTaskID       string     `json:"clickup_task_id,omitempty" gorm:"type:varchar(255)"`
	ClickupURL          string     `json:"clickup_url,omitempty" gorm:"type:text"`
	CreatedAt           time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt           time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for ActionItem
func (ActionItem) TableName() string {
	return "action_items"
}

// NewActionItem creates a new ActionItem entity
func NewActionItem(roomID uuid.UUID, title string) *ActionItem {
	return &ActionItem{
		ID:       uuid.New(),
		RoomID:   roomID,
		Title:    title,
		Type:     "action",
		Priority: "medium",
		Status:   "pending",
	}
}

// ActionItemType constants
const (
	ActionItemTypeAction   = "action"
	ActionItemTypeDecision = "decision"
	ActionItemTypeQuestion = "question"
	ActionItemTypeFollowUp = "follow_up"
	ActionItemTypeResearch = "research"
)

// ActionItemPriority constants
const (
	ActionItemPriorityLow    = "low"
	ActionItemPriorityMedium = "medium"
	ActionItemPriorityHigh   = "high"
	ActionItemPriorityUrgent = "urgent"
)

// ActionItemStatus constants
const (
	ActionItemStatusPending    = "pending"
	ActionItemStatusInProgress = "in_progress"
	ActionItemStatusCompleted  = "completed"
	ActionItemStatusCancelled  = "cancelled"
	ActionItemStatusBlocked    = "blocked"
)
