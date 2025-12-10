package entities

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AIJobStatus represents the status of an AI processing job
type AIJobStatus string

const (
	AIJobStatusPending    AIJobStatus = "pending"    // Waiting to be submitted to AssemblyAI
	AIJobStatusSubmitted  AIJobStatus = "submitted"  // Submitted to AssemblyAI, waiting for transcript
	AIJobStatusProcessing AIJobStatus = "processing" // Being processed by Groq for analysis
	AIJobStatusCompleted  AIJobStatus = "completed"  // All processing done
	AIJobStatusFailed     AIJobStatus = "failed"     // Processing failed
	AIJobStatusRetrying   AIJobStatus = "retrying"   // Retrying after failure
	AIJobStatusCancelled  AIJobStatus = "cancelled"  // Job was cancelled
)

// AIJobType represents the type of AI job
type AIJobType string

const (
	AIJobTypeTranscription AIJobType = "transcription" // Speech to text
	AIJobTypeAnalysis      AIJobType = "analysis"      // LLM analysis
	AIJobTypeReportGen     AIJobType = "report_gen"    // Report generation
)

// AIJob represents an AI processing job for a meeting
type AIJob struct {
	ID            uuid.UUID   `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	MeetingID     uuid.UUID   `json:"meeting_id" gorm:"type:uuid;not null;index"`
	JobType       AIJobType   `json:"job_type" gorm:"type:varchar(50);not null;index"`
	Status        AIJobStatus `json:"status" gorm:"type:varchar(50);not null;index;default:'pending'"`
	ExternalJobID *string     `json:"external_job_id,omitempty" gorm:"type:varchar(255);index"` // AssemblyAI transcript ID (nullable)
	RecordingURL  string      `json:"recording_url" gorm:"type:text;not null"`
	TranscriptID  *uuid.UUID  `json:"transcript_id,omitempty" gorm:"type:uuid;index"`

	// Processing details
	StartedAt   *time.Time `json:"started_at,omitempty" gorm:"type:timestamp"`
	CompletedAt *time.Time `json:"completed_at,omitempty" gorm:"type:timestamp"`
	RetryCount  int        `json:"retry_count" gorm:"type:integer;default:0"`
	MaxRetries  int        `json:"max_retries" gorm:"type:integer;default:3"`
	LastError   *string    `json:"last_error,omitempty" gorm:"type:text"`

	// Metadata
	Metadata AIJobMetadata `json:"metadata,omitempty" gorm:"type:jsonb;serializer:json"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// AIJobMetadata stores additional metadata for AI jobs
type AIJobMetadata struct {
	DurationSeconds  int                    `json:"duration_seconds,omitempty"`
	Language         string                 `json:"language,omitempty"`
	SpeakerCount     int                    `json:"speaker_count,omitempty"`
	ProcessingTimeMs int64                  `json:"processing_time_ms,omitempty"`
	ErrorDetails     map[string]interface{} `json:"error_details,omitempty"`
	WebhookAttempts  int                    `json:"webhook_attempts,omitempty"`
}

// Scan implements sql.Scanner interface for GORM
func (m *AIJobMetadata) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, &m)
}

// Value implements driver.Valuer interface for GORM
func (m AIJobMetadata) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// NewAIJob creates a new AI job
func NewAIJob(meetingID uuid.UUID, jobType AIJobType, recordingURL string) *AIJob {
	return &AIJob{
		ID:           uuid.New(),
		MeetingID:    meetingID,
		JobType:      jobType,
		Status:       AIJobStatusPending,
		RecordingURL: recordingURL,
		RetryCount:   0,
		MaxRetries:   3,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// IsRetryable checks if job can be retried
func (j *AIJob) IsRetryable() bool {
	return j.RetryCount < j.MaxRetries && j.Status == AIJobStatusFailed
}

// CanBeSubmitted checks if job is ready to be submitted
func (j *AIJob) CanBeSubmitted() bool {
	return j.Status == AIJobStatusPending || (j.Status == AIJobStatusFailed && j.IsRetryable())
}

// MarkAsSubmitted marks job as submitted to external service
func (j *AIJob) MarkAsSubmitted(externalJobID string) {
	j.Status = AIJobStatusSubmitted
	j.ExternalJobID = &externalJobID
	now := time.Now()
	j.StartedAt = &now
	j.UpdatedAt = now
}

// MarkAsProcessing marks job as being processed
func (j *AIJob) MarkAsProcessing() {
	j.Status = AIJobStatusProcessing
	j.UpdatedAt = time.Now()
}

// MarkAsCompleted marks job as completed successfully
func (j *AIJob) MarkAsCompleted(transcriptID *uuid.UUID) {
	j.Status = AIJobStatusCompleted
	j.TranscriptID = transcriptID
	now := time.Now()
	j.CompletedAt = &now
	j.UpdatedAt = now
}

// MarkAsFailed marks job as failed with error message
func (j *AIJob) MarkAsFailed(errMsg string) {
	j.Status = AIJobStatusFailed
	j.LastError = &errMsg
	j.UpdatedAt = time.Now()
}

// IncrementRetry increments retry count and marks for retry
func (j *AIJob) IncrementRetry(errMsg string) {
	j.RetryCount++
	j.Status = AIJobStatusRetrying
	j.LastError = &errMsg
	j.UpdatedAt = time.Now()
}

// MarkAsCancelled marks job as cancelled
func (j *AIJob) MarkAsCancelled() {
	j.Status = AIJobStatusCancelled
	now := time.Now()
	j.CompletedAt = &now
	j.UpdatedAt = now
}

// TableName specifies the table name for GORM
func (AIJob) TableName() string {
	return "ai_jobs"
}
