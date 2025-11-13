package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
	"github.com/johnquangdev/meeting-assistant/internal/domain/repositories"
)

// participantRepository implements the ParticipantRepository interface
type participantRepository struct {
	db *gorm.DB
}

// NewParticipantRepository creates a new participant repository
func NewParticipantRepository(db *gorm.DB) repositories.ParticipantRepository {
	return &participantRepository{db: db}
}

// Create creates a new participant record
func (r *participantRepository) Create(ctx context.Context, participant *entities.Participant) error {
	return r.db.WithContext(ctx).Create(participant).Error
}

// FindByID retrieves a participant by ID
func (r *participantRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.Participant, error) {
	var participant entities.Participant
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Room").
		Where("id = ?", id).
		First(&participant).Error

	if err != nil {
		return nil, err
	}
	return &participant, nil
}

// FindByRoomAndUser retrieves a participant by room and user ID
func (r *participantRepository) FindByRoomAndUser(ctx context.Context, roomID, userID uuid.UUID) (*entities.Participant, error) {
	var participant entities.Participant
	err := r.db.WithContext(ctx).
		Preload("User").
		Preload("Room").
		Where("room_id = ? AND user_id = ?", roomID, userID).
		First(&participant).Error

	if err != nil {
		return nil, err
	}
	return &participant, nil
}

// Update updates an existing participant
func (r *participantRepository) Update(ctx context.Context, participant *entities.Participant) error {
	return r.db.WithContext(ctx).Save(participant).Error
}

// Delete deletes a participant record
func (r *participantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entities.Participant{}, id).Error
}

// FindByRoomID retrieves all participants in a room
func (r *participantRepository) FindByRoomID(ctx context.Context, roomID uuid.UUID) ([]*entities.Participant, error) {
	var participants []*entities.Participant
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("room_id = ?", roomID).
		Order("joined_at ASC").
		Find(&participants).Error
	return participants, err
}

// FindActiveByRoomID retrieves all active participants in a room
func (r *participantRepository) FindActiveByRoomID(ctx context.Context, roomID uuid.UUID) ([]*entities.Participant, error) {
	var participants []*entities.Participant
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("room_id = ? AND status = ? AND left_at IS NULL", roomID, entities.ParticipantStatusJoined).
		Order("joined_at ASC").
		Find(&participants).Error
	return participants, err
}

// FindByUserID retrieves all participant records for a user
func (r *participantRepository) FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entities.Participant, error) {
	var participants []*entities.Participant
	query := r.db.WithContext(ctx).
		Preload("Room").
		Where("user_id = ?", userID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&participants).Error
	return participants, err
}

// CountByRoomID counts participants in a room
func (r *participantRepository) CountByRoomID(ctx context.Context, roomID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entities.Participant{}).
		Where("room_id = ?", roomID).
		Count(&count).Error
	return count, err
}

// CountActiveByRoomID counts active participants in a room
func (r *participantRepository) CountActiveByRoomID(ctx context.Context, roomID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entities.Participant{}).
		Where("room_id = ? AND status = ? AND left_at IS NULL", roomID, entities.ParticipantStatusJoined).
		Count(&count).Error
	return count, err
}

// IsUserInRoom checks if a user is currently in a room
func (r *participantRepository) IsUserInRoom(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entities.Participant{}).
		Where("room_id = ? AND user_id = ? AND status = ? AND left_at IS NULL", roomID, userID, entities.ParticipantStatusJoined).
		Count(&count).Error
	return count > 0, err
}

// FindHostByRoomID retrieves the host participant of a room
func (r *participantRepository) FindHostByRoomID(ctx context.Context, roomID uuid.UUID) (*entities.Participant, error) {
	var participant entities.Participant
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("room_id = ? AND role = ?", roomID, entities.ParticipantRoleHost).
		First(&participant).Error

	if err != nil {
		return nil, err
	}
	return &participant, nil
}

// UpdateStatus updates participant status
func (r *participantRepository) UpdateStatus(ctx context.Context, participantID uuid.UUID, status entities.ParticipantStatus) error {
	return r.db.WithContext(ctx).
		Model(&entities.Participant{}).
		Where("id = ?", participantID).
		Update("status", status).
		Error
}

// MarkAsJoined marks a participant as joined
func (r *participantRepository) MarkAsJoined(ctx context.Context, participantID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&entities.Participant{}).
		Where("id = ?", participantID).
		Updates(map[string]interface{}{
			"status":    entities.ParticipantStatusJoined,
			"joined_at": gorm.Expr("NOW()"),
		}).
		Error
}

// MarkAsLeft marks a participant as left
func (r *participantRepository) MarkAsLeft(ctx context.Context, participantID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&entities.Participant{}).
		Where("id = ?", participantID).
		Updates(map[string]interface{}{
			"status":  entities.ParticipantStatusLeft,
			"left_at": gorm.Expr("NOW()"),
		}).
		Error
}

// Remove marks a participant as removed
func (r *participantRepository) Remove(ctx context.Context, participantID, removedBy uuid.UUID, reason string) error {
	return r.db.WithContext(ctx).
		Model(&entities.Participant{}).
		Where("id = ?", participantID).
		Updates(map[string]interface{}{
			"status":         entities.ParticipantStatusRemoved,
			"is_removed":     true,
			"removed_by":     removedBy,
			"removal_reason": reason,
			"left_at":        gorm.Expr("NOW()"),
		}).
		Error
}

// PromoteToHost promotes a participant to host
func (r *participantRepository) PromoteToHost(ctx context.Context, participantID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&entities.Participant{}).
		Where("id = ?", participantID).
		Updates(map[string]interface{}{
			"role":            entities.ParticipantRoleHost,
			"can_record":      true,
			"can_mute_others": true,
		}).
		Error
}

// UpdateRole updates participant role
func (r *participantRepository) UpdateRole(ctx context.Context, participantID uuid.UUID, role entities.ParticipantRole) error {
	updates := map[string]interface{}{
		"role": role,
	}

	// Set permissions based on role
	if role == entities.ParticipantRoleHost || role == entities.ParticipantRoleCoHost {
		updates["can_record"] = true
		updates["can_mute_others"] = true
	}

	return r.db.WithContext(ctx).
		Model(&entities.Participant{}).
		Where("id = ?", participantID).
		Updates(updates).
		Error
}
