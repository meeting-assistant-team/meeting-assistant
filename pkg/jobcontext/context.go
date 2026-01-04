package jobcontext

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type KeyContext string

var (
	keyJobID        KeyContext = "job_id"
	keyJobType      KeyContext = "job_type"
	keyWorkerID     KeyContext = "worker_id"
	keyRetryAttempt KeyContext = "retry_attempt"
	keyJobStartTime KeyContext = "job_start_time"
	keyMaxRetries   KeyContext = "max_retries"
)

// JobMetadata holds metadata for a job execution
type JobMetadata struct {
	JobID        uuid.UUID
	JobType      string
	WorkerID     int
	RetryAttempt int
	MaxRetries   int
	StartTime    time.Time
}

// JobBegin initializes a job context with metadata and timeout
// Creates a derived context from background with 5 minute timeout
func JobBegin(parentCtx context.Context, jobID uuid.UUID, jobType string, workerID int) (context.Context, context.CancelFunc) {
	// Create context with timeout to prevent infinite hanging
	ctx, cancel := context.WithTimeout(parentCtx, 5*time.Minute)

	// Set job metadata
	ctx = context.WithValue(ctx, keyJobID, jobID)
	ctx = context.WithValue(ctx, keyJobType, jobType)
	ctx = context.WithValue(ctx, keyWorkerID, workerID)
	ctx = context.WithValue(ctx, keyRetryAttempt, 0)
	ctx = context.WithValue(ctx, keyMaxRetries, 3)
	ctx = context.WithValue(ctx, keyJobStartTime, time.Now())

	return ctx, cancel
}

// JobEnd executes the job function with automatic commit/rollback and retry logic
// Returns error if job fails after all retries
func JobEnd(ctx context.Context, jobFunc func(context.Context) error) error {
	var (
		err        error
		maxRetries = GetMaxRetries(ctx)
		attempt    = GetRetryAttempt(ctx)
	)

	for attempt < maxRetries {
		// Update retry attempt in context
		ctx = SetRetryAttempt(ctx, attempt)

		// Execute job function with panic recovery
		func(ctx context.Context) {
			defer func() {
				if p := recover(); p != nil {
					err = fmt.Errorf("panic recovered: %v", p)
				}
			}()

			// Check if context was cancelled before execution
			if ctx.Err() != nil {
				err = fmt.Errorf("context cancelled before job execution: %w", ctx.Err())
				return
			}

			err = jobFunc(ctx)
		}(ctx)

		// Job succeeded
		if err == nil {
			return nil
		}

		// Check if error is retryable
		if !IsRetryableError(err) {
			return fmt.Errorf("non-retryable error: %w", err)
		}

		// Increment attempt
		attempt++

		// Check if we've exhausted retries
		if attempt >= maxRetries {
			return fmt.Errorf("max retries (%d) exceeded: %w", maxRetries, err)
		}

		// Exponential backoff: 2^attempt * 5 seconds
		backoff := time.Duration(1<<uint(attempt)) * 5 * time.Second

		// Don't backoff if context is already cancelled
		if ctx.Err() != nil {
			return fmt.Errorf("context cancelled during retry: %w", ctx.Err())
		}

		time.Sleep(backoff)
	}

	return fmt.Errorf("job failed after %d attempts: %w", maxRetries, err)
}

// GetJobID extracts job ID from context
func GetJobID(ctx context.Context) (uuid.UUID, bool) {
	jobID, ok := ctx.Value(keyJobID).(uuid.UUID)
	return jobID, ok
}

// GetJobType extracts job type from context
func GetJobType(ctx context.Context) (string, bool) {
	jobType, ok := ctx.Value(keyJobType).(string)
	return jobType, ok
}

// GetWorkerID extracts worker ID from context
func GetWorkerID(ctx context.Context) int {
	workerID, ok := ctx.Value(keyWorkerID).(int)
	if !ok {
		return -1
	}
	return workerID
}

// GetRetryAttempt extracts current retry attempt from context
func GetRetryAttempt(ctx context.Context) int {
	attempt, ok := ctx.Value(keyRetryAttempt).(int)
	if !ok {
		return 0
	}
	return attempt
}

// SetRetryAttempt updates retry attempt in context
func SetRetryAttempt(ctx context.Context, attempt int) context.Context {
	return context.WithValue(ctx, keyRetryAttempt, attempt)
}

// GetMaxRetries extracts max retries from context
func GetMaxRetries(ctx context.Context) int {
	maxRetries, ok := ctx.Value(keyMaxRetries).(int)
	if !ok {
		return 3 // default
	}
	return maxRetries
}

// SetMaxRetries updates max retries in context
func SetMaxRetries(ctx context.Context, maxRetries int) context.Context {
	return context.WithValue(ctx, keyMaxRetries, maxRetries)
}

// GetJobStartTime extracts job start time from context
func GetJobStartTime(ctx context.Context) (time.Time, bool) {
	startTime, ok := ctx.Value(keyJobStartTime).(time.Time)
	return startTime, ok
}

// SetWorkerMetadata updates worker metadata in context
func SetWorkerMetadata(ctx context.Context, workerID int, attempt int) context.Context {
	ctx = context.WithValue(ctx, keyWorkerID, workerID)
	ctx = context.WithValue(ctx, keyRetryAttempt, attempt)
	return ctx
}

// GetJobMetadata extracts all job metadata from context
func GetJobMetadata(ctx context.Context) *JobMetadata {
	jobID, _ := GetJobID(ctx)
	jobType, _ := GetJobType(ctx)
	startTime, _ := GetJobStartTime(ctx)

	return &JobMetadata{
		JobID:        jobID,
		JobType:      jobType,
		WorkerID:     GetWorkerID(ctx),
		RetryAttempt: GetRetryAttempt(ctx),
		MaxRetries:   GetMaxRetries(ctx),
		StartTime:    startTime,
	}
}

// IsRetryableError checks if an error should trigger a retry
// Retryable errors include: network errors, timeouts, deadlocks, rate limits
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Context errors (timeout, cancelled)
	if strings.Contains(errStr, "context deadline exceeded") ||
		strings.Contains(errStr, "context canceled") {
		return true
	}

	// Network errors
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "network unreachable") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "i/o timeout") {
		return true
	}

	// Database deadlock/lock errors (Postgres)
	if strings.Contains(errStr, "deadlock") ||
		strings.Contains(errStr, "40001") || // serialization_failure
		strings.Contains(errStr, "40p01") { // deadlock_detected
		return true
	}

	// API rate limiting
	if strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "too many requests") ||
		strings.Contains(errStr, "429") {
		return true
	}

	// Server errors (5xx)
	if strings.Contains(errStr, "status 5") ||
		strings.Contains(errStr, "internal server error") ||
		strings.Contains(errStr, "service unavailable") ||
		strings.Contains(errStr, "bad gateway") {
		return true
	}

	// Temporary failures
	if strings.Contains(errStr, "temporary failure") ||
		strings.Contains(errStr, "try again") {
		return true
	}

	return false
}

// IsNonRetryableError checks if an error should NOT trigger a retry
func IsNonRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Client errors (4xx except 429)
	if strings.Contains(errStr, "400") ||
		strings.Contains(errStr, "401") ||
		strings.Contains(errStr, "403") ||
		strings.Contains(errStr, "404") ||
		strings.Contains(errStr, "invalid") ||
		strings.Contains(errStr, "bad request") {
		return true
	}

	// Data validation errors
	if strings.Contains(errStr, "validation failed") ||
		strings.Contains(errStr, "malformed") ||
		strings.Contains(errStr, "parse error") {
		return true
	}

	return false
}

// CalculateBackoff calculates exponential backoff duration
func CalculateBackoff(attempt int, baseDelay time.Duration) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	// 2^attempt * baseDelay, max 60 seconds
	backoff := time.Duration(1<<uint(attempt)) * baseDelay

	maxBackoff := 60 * time.Second
	if backoff > maxBackoff {
		backoff = maxBackoff
	}

	return backoff
}
