package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
)

// TranscriptRepository handles transcript data operations
type TranscriptRepository struct {
	db *gorm.DB
}

// NewTranscriptRepository creates a new transcript repository
func NewTranscriptRepository(db *gorm.DB) *TranscriptRepository {
	return &TranscriptRepository{db: db}
}

// CreateTranscript creates a new transcript
func (r *TranscriptRepository) CreateTranscript(ctx context.Context, transcript *entities.Transcript) error {
	if transcript == nil {
		return errors.New("transcript cannot be nil")
	}
	return r.db.WithContext(ctx).Create(transcript).Error
}

// GetTranscriptByID retrieves a transcript by ID
func (r *TranscriptRepository) GetTranscriptByID(ctx context.Context, id uuid.UUID) (*entities.Transcript, error) {
	var transcript entities.Transcript
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&transcript).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &transcript, nil
}

// GetTranscriptByMeetingID retrieves a transcript by meeting ID
func (r *TranscriptRepository) GetTranscriptByMeetingID(ctx context.Context, meetingID uuid.UUID) (*entities.Transcript, error) {
	var transcript entities.Transcript
	if err := r.db.WithContext(ctx).Where("meeting_id = ?", meetingID).First(&transcript).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &transcript, nil
}

// UpdateTranscript updates a transcript
func (r *TranscriptRepository) UpdateTranscript(ctx context.Context, transcript *entities.Transcript) error {
	if transcript == nil {
		return errors.New("transcript cannot be nil")
	}
	return r.db.WithContext(ctx).
		Model(&entities.Transcript{}).
		Where("id = ?", transcript.ID).
		Save(transcript).Error
}

// StoreTranscriptData stores the full transcript data (called after AssemblyAI webhook)
func (r *TranscriptRepository) StoreTranscriptData(ctx context.Context, transcriptID uuid.UUID, data map[string]interface{}) error {
	// Convert data to JSON
	jsonData, err := datatypes.JSONType[map[string]interface{}]{}.MarshalJSON()
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).
		Model(&entities.Transcript{}).
		Where("id = ?", transcriptID).
		Updates(map[string]interface{}{
			"raw_data":   jsonData,
			"updated_at": time.Now(),
		}).Error
}

// DeleteTranscript deletes a transcript
func (r *TranscriptRepository) DeleteTranscript(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entities.Transcript{}, id).Error
}
