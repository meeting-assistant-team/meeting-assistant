package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
	"github.com/johnquangdev/meeting-assistant/internal/domain/repositories"
)

// roomRepository implements the RoomRepository interface
type roomRepository struct {
	db *gorm.DB
}

// NewRoomRepository creates a new room repository
func NewRoomRepository(db *gorm.DB) repositories.RoomRepository {
	return &roomRepository{db: db}
}

// Create creates a new room
func (r *roomRepository) Create(ctx context.Context, room *entities.Room) error {
	return r.db.WithContext(ctx).Create(room).Error
}

// FindByID retrieves a room by its ID
func (r *roomRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.Room, error) {
	var room entities.Room
	err := r.db.WithContext(ctx).
		Preload("Host").
		Where("id = ?", id).
		First(&room).Error

	if err != nil {
		return nil, err
	}
	return &room, nil
}

// FindBySlug retrieves a room by its slug
func (r *roomRepository) FindBySlug(ctx context.Context, slug string) (*entities.Room, error) {
	var room entities.Room
	err := r.db.WithContext(ctx).
		Preload("Host").
		Where("slug = ?", slug).
		First(&room).Error

	if err != nil {
		return nil, err
	}
	return &room, nil
}

// FindByLivekitName retrieves a room by its LiveKit room name
func (r *roomRepository) FindByLivekitName(ctx context.Context, livekitName string) (*entities.Room, error) {
	var room entities.Room
	err := r.db.WithContext(ctx).
		Preload("Host").
		Where("livekit_room_name = ?", livekitName).
		First(&room).Error

	if err != nil {
		return nil, err
	}
	return &room, nil
}

// Update updates an existing room
func (r *roomRepository) Update(ctx context.Context, room *entities.Room) error {
	return r.db.WithContext(ctx).Save(room).Error
}

// Delete soft deletes a room
func (r *roomRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entities.Room{}, id).Error
}

// List retrieves rooms with filters and pagination
func (r *roomRepository) List(ctx context.Context, filters repositories.RoomFilters) ([]*entities.Room, int64, error) {
	var rooms []*entities.Room
	var total int64

	query := r.db.WithContext(ctx).Model(&entities.Room{}).Preload("Host")

	// Apply filters
	if filters.Type != nil {
		query = query.Where("type = ?", *filters.Type)
	}
	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}
	if filters.HostID != nil {
		query = query.Where("host_id = ?", *filters.HostID)
	}
	if filters.Search != "" {
		searchPattern := fmt.Sprintf("%%%s%%", filters.Search)
		query = query.Where("name ILIKE ? OR description ILIKE ?", searchPattern, searchPattern)
	}
	if len(filters.Tags) > 0 {
		query = query.Where("tags @> ?", filters.Tags)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	sortBy := "created_at"
	if filters.SortBy != "" {
		sortBy = filters.SortBy
	}
	sortOrder := "DESC"
	if filters.SortOrder != "" {
		sortOrder = filters.SortOrder
	}
	query = query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))

	// Apply pagination
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	err := query.Find(&rooms).Error
	return rooms, total, err
}

// FindByHostID retrieves all rooms hosted by a user
func (r *roomRepository) FindByHostID(ctx context.Context, hostID uuid.UUID, limit, offset int) ([]*entities.Room, error) {
	var rooms []*entities.Room
	query := r.db.WithContext(ctx).
		Where("host_id = ?", hostID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&rooms).Error
	return rooms, err
}

// FindActiveRooms retrieves all currently active rooms
func (r *roomRepository) FindActiveRooms(ctx context.Context) ([]*entities.Room, error) {
	var rooms []*entities.Room
	err := r.db.WithContext(ctx).
		Preload("Host").
		Where("status = ?", entities.RoomStatusActive).
		Order("started_at DESC").
		Find(&rooms).Error
	return rooms, err
}

// FindScheduledRooms retrieves all scheduled rooms
func (r *roomRepository) FindScheduledRooms(ctx context.Context) ([]*entities.Room, error) {
	var rooms []*entities.Room
	err := r.db.WithContext(ctx).
		Preload("Host").
		Where("status = ?", entities.RoomStatusScheduled).
		Order("scheduled_start_time ASC").
		Find(&rooms).Error
	return rooms, err
}

// IncrementParticipantCount increases the participant count
func (r *roomRepository) IncrementParticipantCount(ctx context.Context, roomID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&entities.Room{}).
		Where("id = ?", roomID).
		UpdateColumn("current_participants", gorm.Expr("current_participants + 1")).
		Error
}

// DecrementParticipantCount decreases the participant count
func (r *roomRepository) DecrementParticipantCount(ctx context.Context, roomID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&entities.Room{}).
		Where("id = ? AND current_participants > 0", roomID).
		UpdateColumn("current_participants", gorm.Expr("current_participants - 1")).
		Error
}

// UpdateStatus updates the room status
func (r *roomRepository) UpdateStatus(ctx context.Context, roomID uuid.UUID, status entities.RoomStatus) error {
	return r.db.WithContext(ctx).
		Model(&entities.Room{}).
		Where("id = ?", roomID).
		Update("status", status).
		Error
}

// EndRoom marks a room as ended and calculates duration
func (r *roomRepository) EndRoom(ctx context.Context, roomID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&entities.Room{}).
		Where("id = ?", roomID).
		Updates(map[string]interface{}{
			"status":   entities.RoomStatusEnded,
			"ended_at": gorm.Expr("NOW()"),
		}).
		Error
}
