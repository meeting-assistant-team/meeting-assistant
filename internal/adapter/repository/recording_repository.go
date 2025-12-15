package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
)

// RecordingRepository handles recording data operations
type RecordingRepository struct {
	db *gorm.DB
}

// NewRecordingRepository creates a new recording repository
func NewRecordingRepository(db *gorm.DB) *RecordingRepository {
	return &RecordingRepository{db: db}
}

// Create creates a new recording
func (r *RecordingRepository) Create(ctx context.Context, recording *entities.Recording) error {
	if recording == nil {
		return errors.New("recording cannot be nil")
	}
	return r.db.WithContext(ctx).Create(recording).Error
}

// FindByID retrieves a recording by ID
func (r *RecordingRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.Recording, error) {
	var recording entities.Recording
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&recording).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &recording, nil
}

// FindByRoomID retrieves all recordings for a room
func (r *RecordingRepository) FindByRoomID(ctx context.Context, roomID uuid.UUID) ([]*entities.Recording, error) {
	var recordings []*entities.Recording
	if err := r.db.WithContext(ctx).
		Where("room_id = ?", roomID).
		Order("started_at DESC").
		Find(&recordings).Error; err != nil {
		return nil, err
	}
	return recordings, nil
}

// FindByEgressID retrieves a recording by LiveKit egress ID
func (r *RecordingRepository) FindByEgressID(ctx context.Context, egressID string) (*entities.Recording, error) {
	var recording entities.Recording
	if err := r.db.WithContext(ctx).
		Where("livekit_egress_id = ?", egressID).
		First(&recording).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &recording, nil
}

// Update updates a recording
func (r *RecordingRepository) Update(ctx context.Context, recording *entities.Recording) error {
	if recording == nil {
		return errors.New("recording cannot be nil")
	}
	return r.db.WithContext(ctx).Save(recording).Error
}

// UpdateStatus updates recording status
func (r *RecordingRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entities.RecordingStatus) error {
	return r.db.WithContext(ctx).
		Model(&entities.Recording{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// Delete deletes a recording
func (r *RecordingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entities.Recording{}, id).Error
}

// FindPendingProcessing finds recordings that need processing (for polling fallback)
func (r *RecordingRepository) FindPendingProcessing(ctx context.Context) ([]*entities.Recording, error) {
	var recordings []*entities.Recording
	if err := r.db.WithContext(ctx).
		Where("status = ? OR status = ?", entities.RecordingStatusRecording, entities.RecordingStatusProcessing).
		Where("created_at > NOW() - INTERVAL '24 hours'"). // Only recent recordings
		Order("created_at ASC").
		Find(&recordings).Error; err != nil {
		return nil, err
	}
	return recordings, nil
}
