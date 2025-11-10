# Use Case Layer

**Layer 2 - Application Business Rules**

## Purpose
Contains application-specific business logic. Orchestrates the flow of data between entities and external systems. Implements the actual features of the application.

## Structure

### `auth/`
Authentication and authorization use cases.

**Files:**
- `login.go` - Google OAuth2 login flow
- `refresh_token.go` - Token refresh logic
- `logout.go` - Logout and session revocation
- `get_current_user.go` - Get authenticated user info

### `room/`
Meeting room management use cases.

**Files:**
- `create_room.go` - Create new meeting room
- `join_room.go` - Join existing room
- `leave_room.go` - Leave room
- `end_room.go` - End meeting (host only)
- `list_rooms.go` - List user's rooms
- `update_room.go` - Update room settings

### `recording/`
Recording management use cases.

**Files:**
- `start_recording.go` - Start recording meeting
- `stop_recording.go` - Stop and save recording
- `get_recording.go` - Retrieve recording info
- `list_recordings.go` - List room recordings

### `ai/`
AI analysis use cases.

**Files:**
- `transcribe_audio.go` - Convert audio to text using Whisper
- `analyze_meeting.go` - Generate meeting analysis with GPT-4
- `extract_action_items.go` - Extract action items
- `generate_summary.go` - Generate meeting summary

### `report/`
Report generation use cases.

**Files:**
- `generate_report.go` - Generate meeting report
- `export_pdf.go` - Export report to PDF
- `get_dashboard_data.go` - Get dashboard statistics

### `integration/`
Third-party integration use cases.

**Files:**
- `sync_to_clickup.go` - Sync action items to ClickUp

## Example Use Case

```go
// usecase/room/create_room.go
package room

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
	"github.com/johnquangdev/meeting-assistant/internal/domain/repositories"
)

type CreateRoomInput struct {
	HostID      uuid.UUID
	Name        string
	Description string
	IsPrivate   bool
}

type CreateRoomOutput struct {
	Room            *entities.Room
	LiveKitToken    string
	LiveKitURL      string
}

type CreateRoomUseCase struct {
	roomRepo      repositories.RoomRepository
	livekitClient LiveKitClient
}

func NewCreateRoomUseCase(
	roomRepo repositories.RoomRepository,
	livekitClient LiveKitClient,
) *CreateRoomUseCase {
	return &CreateRoomUseCase{
		roomRepo:      roomRepo,
		livekitClient: livekitClient,
	}
}

func (uc *CreateRoomUseCase) Execute(ctx context.Context, input CreateRoomInput) (*CreateRoomOutput, error) {
	// 1. Validate input
	if input.Name == "" {
		return nil, ErrRoomNameRequired
	}

	// 2. Create room entity
	room := &entities.Room{
		ID:          uuid.New(),
		Name:        input.Name,
		Description: input.Description,
		HostID:      input.HostID,
		Status:      entities.RoomStatusWaiting,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 3. Save to database
	if err := uc.roomRepo.Create(ctx, room); err != nil {
		return nil, err
	}

	// 4. Create LiveKit room
	livekitRoom, err := uc.livekitClient.CreateRoom(ctx, room.ID.String())
	if err != nil {
		return nil, err
	}

	// 5. Generate LiveKit token for host
	token, err := uc.livekitClient.GenerateToken(ctx, room.ID.String(), input.HostID.String(), true)
	if err != nil {
		return nil, err
	}

	return &CreateRoomOutput{
		Room:         room,
		LiveKitToken: token,
		LiveKitURL:   uc.livekitClient.GetURL(),
	}, nil
}

// LiveKitClient interface (will be implemented in infrastructure layer)
type LiveKitClient interface {
	CreateRoom(ctx context.Context, roomID string) error
	GenerateToken(ctx context.Context, roomID, userID string, isHost bool) (string, error)
	GetURL() string
}
```

## Rules

✅ **DO:**
- Orchestrate business logic flow
- Depend only on domain layer interfaces
- Define input/output DTOs
- Handle application-specific errors
- Coordinate between multiple repositories
- Call external services through interfaces

❌ **DON'T:**
- Depend on HTTP handlers or frameworks
- Import database drivers directly
- Import external service SDKs directly
- Handle HTTP requests/responses
- Know about JSON, XML, or other serialization formats

## Dependencies
- **Depends on:** Domain Layer (entities and repository interfaces)
- **Depended by:** Adapter Layer (handlers)

## Testing

Use cases should be tested with mocked repositories:

```go
// usecase/room/create_room_test.go
func TestCreateRoomUseCase_Execute(t *testing.T) {
	mockRepo := new(MockRoomRepository)
	mockLiveKit := new(MockLivekitClient)
	
	uc := NewCreateRoomUseCase(mockRepo, mockLiveKit)
	
	input := CreateRoomInput{
		HostID: uuid.New(),
		Name:   "Test Room",
	}
	
	mockRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockLiveKit.On("CreateRoom", mock.Anything, mock.Anything).Return(nil)
	mockLiveKit.On("GenerateToken", mock.Anything, mock.Anything, mock.Anything, true).Return("token", nil)
	
	output, err := uc.Execute(context.Background(), input)
	
	assert.NoError(t, err)
	assert.NotNil(t, output.Room)
	assert.Equal(t, input.Name, output.Room.Name)
	mockRepo.AssertExpectations(t)
	mockLiveKit.AssertExpectations(t)
}
```
