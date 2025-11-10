# Domain Layer

**Layer 1 - Innermost Layer of Clean Architecture**

## Purpose
Contains the core business logic and rules. This layer is the heart of the application and should be completely independent of any external frameworks, databases, or UI.

## Structure

### `entities/`
Business entities representing the core domain models.

**Files:**
- `user.go` - User entity and business rules
- `room.go` - Meeting room entity and state management
- `participant.go` - Participant entity and roles
- `recording.go` - Recording metadata and lifecycle
- `transcript.go` - Transcript content and segments
- `meeting_summary.go` - AI-generated meeting summaries
- `action_item.go` - Extracted action items

**Example:**
```go
// entities/room.go
package entities

import (
	"time"
	"github.com/google/uuid"
)

type RoomStatus string

const (
	RoomStatusWaiting  RoomStatus = "waiting"
	RoomStatusActive   RoomStatus = "active"
	RoomStatusEnded    RoomStatus = "ended"
)

type Room struct {
	ID          uuid.UUID
	Name        string
	Description string
	HostID      uuid.UUID
	Status      RoomStatus
	StartedAt   *time.Time
	EndedAt     *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Business methods
func (r *Room) CanStart() bool {
	return r.Status == RoomStatusWaiting
}

func (r *Room) Start() error {
	if !r.CanStart() {
		return ErrRoomAlreadyStarted
	}
	now := time.Now()
	r.Status = RoomStatusActive
	r.StartedAt = &now
	return nil
}

func (r *Room) End() error {
	if r.Status != RoomStatusActive {
		return ErrRoomNotActive
	}
	now := time.Now()
	r.Status = RoomStatusEnded
	r.EndedAt = &now
	return nil
}
```

### `repositories/`
Repository interfaces defining data access contracts.

**Files:**
- `user_repository.go`
- `room_repository.go`
- `recording_repository.go`
- `transcript_repository.go`
- `session_repository.go`

**Example:**
```go
// repositories/room_repository.go
package repositories

import (
	"context"
	"github.com/google/uuid"
	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
)

type RoomRepository interface {
	Create(ctx context.Context, room *entities.Room) error
	FindByID(ctx context.Context, id uuid.UUID) (*entities.Room, error)
	FindByHostID(ctx context.Context, hostID uuid.UUID) ([]*entities.Room, error)
	Update(ctx context.Context, room *entities.Room) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter RoomFilter) ([]*entities.Room, error)
}

type RoomFilter struct {
	Status   *entities.RoomStatus
	HostID   *uuid.UUID
	Limit    int
	Offset   int
}
```

## Rules

✅ **DO:**
- Define pure business logic
- Keep entities framework-agnostic
- Define repository interfaces (not implementations)
- Use domain-specific types and enums
- Implement business validation rules

❌ **DON'T:**
- Import any external frameworks (Echo, GORM, etc.)
- Depend on use case, adapter, or infrastructure layers
- Include database, HTTP, or external service code
- Use concrete implementations (only interfaces)

## Dependencies
- **Depends on:** NOTHING (completely independent)
- **Depended by:** Use Case Layer, Adapter Layer

## Testing
Domain layer should have the highest test coverage as it contains critical business logic.

```go
// entities/room_test.go
func TestRoom_Start(t *testing.T) {
	room := &entities.Room{
		ID:     uuid.New(),
		Status: entities.RoomStatusWaiting,
	}
	
	err := room.Start()
	assert.NoError(t, err)
	assert.Equal(t, entities.RoomStatusActive, room.Status)
	assert.NotNil(t, room.StartedAt)
}
```
