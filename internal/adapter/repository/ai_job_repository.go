package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
)

// AIJobRepository handles AI job data operations
type AIJobRepository struct {
	db *gorm.DB
}

// NewAIJobRepository creates a new AI job repository
func NewAIJobRepository(db *gorm.DB) *AIJobRepository {
	return &AIJobRepository{db: db}
}

// CreateAIJob creates a new AI job
func (r *AIJobRepository) CreateAIJob(ctx context.Context, job *entities.AIJob) error {
	if job == nil {
		return errors.New("job cannot be nil")
	}
	return r.db.WithContext(ctx).Create(job).Error
}

// GetAIJobByID retrieves an AI job by ID
func (r *AIJobRepository) GetAIJobByID(ctx context.Context, jobID uuid.UUID) (*entities.AIJob, error) {
	var job entities.AIJob
	if err := r.db.WithContext(ctx).Where("id = ?", jobID).First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &job, nil
}

// GetAIJobByExternalID retrieves an AI job by external job ID (AssemblyAI transcript ID)
func (r *AIJobRepository) GetAIJobByExternalID(ctx context.Context, externalID string) (*entities.AIJob, error) {
	var job entities.AIJob
	if err := r.db.WithContext(ctx).Where("external_job_id = ?", externalID).First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &job, nil
}

// GetAIJobByMeetingID retrieves the latest AI job for a meeting
func (r *AIJobRepository) GetAIJobByMeetingID(ctx context.Context, meetingID uuid.UUID, jobType entities.AIJobType) (*entities.AIJob, error) {
	var job entities.AIJob
	query := r.db.WithContext(ctx).Where("meeting_id = ?", meetingID)
	if jobType != "" {
		query = query.Where("job_type = ?", jobType)
	}
	if err := query.Order("created_at DESC").First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &job, nil
}

// ListAIJobsByMeetingID retrieves all AI jobs for a meeting
func (r *AIJobRepository) ListAIJobsByMeetingID(ctx context.Context, meetingID uuid.UUID) ([]entities.AIJob, error) {
	var jobs []entities.AIJob
	if err := r.db.WithContext(ctx).
		Where("meeting_id = ?", meetingID).
		Order("created_at DESC").
		Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

// ListAIJobsByStatus retrieves all AI jobs with a specific status
func (r *AIJobRepository) ListAIJobsByStatus(ctx context.Context, status entities.AIJobStatus, limit int) ([]entities.AIJob, error) {
	var jobs []entities.AIJob
	if limit == 0 {
		limit = 100
	}
	if err := r.db.WithContext(ctx).
		Where("status = ?", status).
		Order("created_at ASC").
		Limit(limit).
		Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

// UpdateAIJobStatus updates the status of an AI job
func (r *AIJobRepository) UpdateAIJobStatus(ctx context.Context, jobID uuid.UUID, status entities.AIJobStatus) error {
	return r.db.WithContext(ctx).
		Model(&entities.AIJob{}).
		Where("id = ?", jobID).
		Update("status", status).Error
}

// UpdateAIJob updates an AI job
func (r *AIJobRepository) UpdateAIJob(ctx context.Context, job *entities.AIJob) error {
	if job == nil {
		return errors.New("job cannot be nil")
	}
	return r.db.WithContext(ctx).
		Model(&entities.AIJob{}).
		Where("id = ?", job.ID).
		Save(job).Error
}

// MarkJobAsSubmitted marks a job as submitted with external ID
func (r *AIJobRepository) MarkJobAsSubmitted(ctx context.Context, jobID uuid.UUID, externalID string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&entities.AIJob{}).
		Where("id = ?", jobID).
		Updates(map[string]interface{}{
			"status":          entities.AIJobStatusSubmitted,
			"external_job_id": externalID,
			"started_at":      now,
			"updated_at":      now,
		}).Error
}

// MarkJobAsCompleted marks a job as completed with transcript ID
func (r *AIJobRepository) MarkJobAsCompleted(ctx context.Context, jobID uuid.UUID, transcriptID *uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&entities.AIJob{}).
		Where("id = ?", jobID).
		Updates(map[string]interface{}{
			"status":        entities.AIJobStatusCompleted,
			"transcript_id": transcriptID,
			"completed_at":  now,
			"updated_at":    now,
		}).Error
}

// MarkJobAsFailed marks a job as failed with error message
func (r *AIJobRepository) MarkJobAsFailed(ctx context.Context, jobID uuid.UUID, errMsg string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&entities.AIJob{}).
		Where("id = ?", jobID).
		Updates(map[string]interface{}{
			"status":     entities.AIJobStatusFailed,
			"last_error": errMsg,
			"updated_at": now,
		}).Error
}

// IncrementRetryCount increments the retry count
func (r *AIJobRepository) IncrementRetryCount(ctx context.Context, jobID uuid.UUID, errMsg string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&entities.AIJob{}).
		Where("id = ?", jobID).
		Updates(map[string]interface{}{
			"retry_count": gorm.Expr("retry_count + 1"),
			"status":      entities.AIJobStatusRetrying,
			"last_error":  errMsg,
			"updated_at":  now,
		}).Error
}

// DeleteAIJob soft deletes an AI job (if using soft delete)
func (r *AIJobRepository) DeleteAIJob(ctx context.Context, jobID uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entities.AIJob{}, jobID).Error
}

// GetFailedJobs retrieves jobs that failed and can be retried
func (r *AIJobRepository) GetFailedJobs(ctx context.Context, limit int) ([]entities.AIJob, error) {
	var jobs []entities.AIJob
	if limit == 0 {
		limit = 10
	}
	if err := r.db.WithContext(ctx).
		Where("status = ? AND retry_count < max_retries", entities.AIJobStatusFailed).
		Order("updated_at ASC").
		Limit(limit).
		Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

// GetJobsForProcessing retrieves jobs that are ready for processing
func (r *AIJobRepository) GetJobsForProcessing(ctx context.Context, limit int) ([]entities.AIJob, error) {
	var jobs []entities.AIJob
	if limit == 0 {
		limit = 10
	}
	if err := r.db.WithContext(ctx).
		Where("status IN ?", []entities.AIJobStatus{entities.AIJobStatusPending, entities.AIJobStatusRetrying}).
		Order("created_at ASC").
		Limit(limit).
		Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

// GetJobsWithoutExternalID retrieves submitted jobs that don't have external ID (shouldn't happen but for recovery)
func (r *AIJobRepository) GetJobsWithoutExternalID(ctx context.Context, limit int) ([]entities.AIJob, error) {
	var jobs []entities.AIJob
	if limit == 0 {
		limit = 10
	}
	if err := r.db.WithContext(ctx).
		Where("status = ? AND (external_job_id IS NULL OR external_job_id = '')", entities.AIJobStatusSubmitted).
		Order("created_at ASC").
		Limit(limit).
		Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}
