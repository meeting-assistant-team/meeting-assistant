# Room Management & Meeting Flow

## Overview

Hệ thống quản lý phòng họp cho phép người dùng tạo, tham gia và quản lý các cuộc họp trực tuyến với audio/video real-time thông qua LiveKit.

## Room Types

### Public Room
- Bất kỳ ai có link đều có thể tham gia
- Không cần phê duyệt
- Hiển thị trong danh sách public rooms

### Private Room
- Chỉ người được mời mới tham gia
- Cần approval từ host
- Yêu cầu access code hoặc invitation token

## Create Room Flow

```mermaid
sequenceDiagram
    participant U as User (Host)
    participant F as Frontend
    participant B as Backend
    participant LK as LiveKit Server
    participant DB as Database
    
    U->>F: Click "Create Meeting"
    F->>F: Show create room form
    U->>F: Fill room details<br/>(name, type, max_participants)
    
    F->>B: POST /api/rooms<br/>{ name, type, settings }
    Note over F,B: Authorization: Bearer {token}
    
    B->>B: Validate user role (host/admin)
    B->>B: Generate room_id (UUID)
    
    B->>LK: Create room via API
    Note over B,LK: POST /twirp/livekit.RoomService/CreateRoom
    LK-->>B: Room created with metadata
    
    B->>DB: INSERT INTO rooms
    Note over DB: room_id, name, host_id,<br/>type, settings, livekit_room_name
    
    B->>DB: INSERT INTO participants
    Note over DB: Host is first participant
    
    B->>B: Generate LiveKit token for host
    Note over B: Token with permissions:<br/>CanPublish, CanSubscribe,<br/>CanPublishData
    
    B-->>F: { room, livekit_token, livekit_url }
    
    F->>F: Navigate to room page
    F->>LK: Connect via WebSocket
    Note over F,LK: Using LiveKit SDK
    
    F-->>U: Show meeting room UI
```

## Join Room Flow

```mermaid
sequenceDiagram
    participant U as User (Participant)
    participant F as Frontend
    participant B as Backend
    participant LK as LiveKit Server
    participant DB as Database
    participant WS as WebSocket (Notifications)
    
    U->>F: Enter room code or click invite link
    F->>B: GET /api/rooms/:id/join
    Note over F,B: Authorization: Bearer {token}
    
    B->>DB: SELECT room WHERE id = ?
    
    alt Room not found
        B-->>F: 404 Not Found
        F-->>U: "Room does not exist"
    else Room found
        B->>DB: Check room capacity
        
        alt Room full
            B-->>F: 400 Bad Request "Room is full"
        else Room available
            alt Private room
                B->>DB: Check if user invited
                alt Not invited
                    B-->>F: 403 Forbidden
                    F-->>U: "Request access"
                end
            end
            
            B->>DB: INSERT INTO participants
            Note over DB: user_id, room_id,<br/>joined_at, role: participant
            
            B->>B: Generate LiveKit token
            Note over B: Participant permissions
            
            B->>DB: UPDATE rooms SET participant_count++
            
            B->>WS: Notify existing participants
            Note over WS: { event: "participant_joined",<br/>user: {...} }
            
            B-->>F: { room, livekit_token, participants }
            
            F->>LK: Connect to LiveKit room
            F->>LK: Enable camera/microphone
            F-->>U: Show meeting UI
        end
    end
```

## Complete Meeting Flow

```mermaid
sequenceDiagram
    participant H as Host
    participant P as Participants
    participant F as Frontend
    participant B as Backend
    participant LK as LiveKit Server
    participant DB as Database
    participant RS as Recording Service
    participant AI as AI Service
    
    Note over H,AI: Meeting in progress
    
    H->>F: Click "End Meeting"
    F->>B: POST /api/rooms/:id/end
    
    B->>LK: Stop all recordings
    LK-->>B: Recording stopped
    
    B->>DB: UPDATE rooms SET status = 'ended'
    B->>DB: UPDATE participants SET left_at = NOW()
    
    B->>WS: Notify all participants
    Note over WS: { event: "meeting_ended" }
    
    WS-->>F: Broadcast to all clients
    F-->>P: Show "Meeting ended" message
    F->>F: Disconnect from LiveKit
    F->>F: Redirect to feedback page
    
    B->>RS: Trigger recording processing
    activate RS
    RS->>LK: Download recording files
    RS->>RS: Merge audio tracks
    RS->>DB: Store recording metadata
    RS->>AI: Queue for transcription
    deactivate RS
    
    activate AI
    AI->>AI: Process audio (Whisper STT)
    AI->>AI: Generate transcript
    AI->>DB: Store transcript
    AI->>AI: Analyze with GPT-4
    AI->>DB: Store summary & action items
    AI->>B: Notify processing complete
    deactivate AI
    
    B->>WS: Notify participants
    Note over WS: "Report ready"
    WS-->>F: Update UI
    F-->>P: Show notification "Report available"
```

## Leave Room Flow

```mermaid
sequenceDiagram
    participant U as User
    participant F as Frontend
    participant B as Backend
    participant LK as LiveKit Server
    participant DB as Database
    participant WS as WebSocket
    
    alt User clicks "Leave"
        U->>F: Click "Leave Meeting"
    else Network disconnection
        F->>F: Detect connection lost
    else Tab closed
        F->>F: beforeunload event
    end
    
    F->>LK: Disconnect from room
    LK->>LK: Remove participant from SFU
    
    F->>B: POST /api/rooms/:id/leave
    Note over F,B: Or use WebSocket if available
    
    B->>DB: UPDATE participants<br/>SET left_at = NOW()
    B->>DB: UPDATE rooms<br/>SET participant_count--
    
    B->>WS: Notify other participants
    Note over WS: { event: "participant_left",<br/>user_id: "..." }
    
    alt User is host and room not empty
        B->>B: Promote another participant to host
        B->>DB: UPDATE participant SET role = 'host'
        B->>WS: Notify new host
    else Last participant leaving
        B->>B: Auto-end meeting
        B->>DB: UPDATE rooms SET status = 'ended'
    end
    
    B-->>F: 200 OK
    F->>F: Clear room state
    F-->>U: Redirect to dashboard
```

## Invite Participant Flow

```mermaid
sequenceDiagram
    participant H as Host
    participant F as Frontend
    participant B as Backend
    participant E as Email Service
    participant DB as Database
    
    H->>F: Click "Invite People"
    F->>F: Show invite dialog
    H->>F: Enter email addresses
    
    F->>B: POST /api/rooms/:id/invite
    Note over F,B: { emails: ["user1@ex.com", "user2@ex.com"] }
    
    loop For each email
        B->>DB: Check if user exists
        
        alt User exists
            B->>DB: INSERT INTO room_invitations
            Note over DB: room_id, user_id, invited_by
            B->>DB: INSERT INTO notifications
        else User not exists
            B->>DB: INSERT INTO pending_invitations
            Note over DB: room_id, email, token
        end
        
        B->>B: Generate invitation link
        Note over B: /rooms/:id/join?token=xxx
        
        B->>E: Send invitation email
        Note over E: Subject: "You're invited to meeting"<br/>Body: Link + meeting details
    end
    
    B-->>F: { sent: 5, failed: 0 }
    F-->>H: Show success message
```

## Recording Control Flow

```mermaid
sequenceDiagram
    participant H as Host
    participant F as Frontend
    participant B as Backend
    participant LK as LiveKit Server
    participant DB as Database
    participant S3 as S3 Storage
    
    H->>F: Click "Start Recording"
    F->>B: POST /api/rooms/:id/recording/start
    
    B->>B: Check host permissions
    B->>LK: Start room recording
    Note over B,LK: POST /twirp/livekit.RecordingService/StartRecording
    
    LK->>LK: Start capturing all tracks
    LK-->>B: { recording_id, status: "active" }
    
    B->>DB: INSERT INTO recordings
    Note over DB: room_id, recording_id,<br/>status: 'recording'
    
    B-->>F: { recording_id, started_at }
    F->>F: Show recording indicator
    F-->>H: Update UI "Recording..."
    
    Note over H,S3: ... Meeting continues ...
    
    H->>F: Click "Stop Recording"
    F->>B: POST /api/rooms/:id/recording/stop
    
    B->>LK: Stop recording
    LK->>LK: Finalize recording file
    LK->>S3: Upload recording
    LK-->>B: { recording_url, duration, file_size }
    
    B->>DB: UPDATE recordings<br/>SET status = 'completed',<br/>file_url = ?, duration = ?
    
    B-->>F: { status: "completed" }
    F->>F: Hide recording indicator
```

## Screen Share Flow

```mermaid
sequenceDiagram
    participant U as User
    participant F as Frontend
    participant B as Backend
    participant LK as LiveKit Server
    participant P as Other Participants
    
    U->>F: Click "Share Screen"
    F->>F: Request screen capture
    Note over F: navigator.mediaDevices.getDisplayMedia()
    
    F->>F: Get screen track
    F->>LK: Publish screen track
    Note over F,LK: LiveKit SDK: room.localParticipant<br/>.publishTrack(screenTrack)
    
    LK->>LK: Add track to SFU
    LK->>P: Notify new track available
    
    P->>LK: Subscribe to screen track
    LK-->>P: Stream screen share
    P->>P: Display shared screen
    
    Note over U,P: ... Screen sharing active ...
    
    alt User stops sharing
        U->>F: Click "Stop Sharing"
        F->>LK: Unpublish track
    else Screen share window closed
        F->>F: Track ended event
        F->>LK: Unpublish track
    end
    
    LK->>P: Notify track removed
    P->>P: Hide screen share display
```

## Room Settings Management

```mermaid
sequenceDiagram
    participant H as Host
    participant F as Frontend
    participant B as Backend
    participant DB as Database
    participant WS as WebSocket
    participant P as Participants
    
    H->>F: Open room settings
    F->>F: Show settings panel
    
    alt Change room permissions
        H->>F: Update setting<br/>(e.g., mute all, lock room)
        F->>B: PATCH /api/rooms/:id/settings
        B->>DB: UPDATE rooms SET settings = ?
        B->>WS: Broadcast settings update
        WS-->>P: Apply new settings
        
    else Mute participant
        H->>F: Click mute on participant
        F->>B: POST /api/rooms/:id/participants/:pid/mute
        B->>WS: Send mute command to specific user
        WS-->>P: Mute audio track
        
    else Remove participant
        H->>F: Click remove participant
        F->>B: DELETE /api/rooms/:id/participants/:pid
        B->>DB: UPDATE participant SET removed = true
        B->>WS: Force disconnect participant
        WS-->>P: Disconnect and show message
        
    else Transfer host role
        H->>F: Promote participant to host
        F->>B: POST /api/rooms/:id/transfer-host
        B->>DB: UPDATE participants<br/>SET role = 'host' or 'participant'
        B->>WS: Notify role changes
        WS-->>P: Update UI permissions
    end
```

## API Endpoints

### Room Management

```yaml
# Create Room
POST /api/rooms
  Headers: Authorization: Bearer {token}
  Body:
    name: string (required)
    description: string
    type: "public" | "private"
    max_participants: number (default: 10)
    settings:
      enable_recording: boolean
      enable_chat: boolean
      require_approval: boolean (for private)
  Response:
    room: RoomObject
    livekit_token: string
    livekit_url: string

# Get Room Details
GET /api/rooms/:id
  Response:
    room: RoomObject
    participants: ParticipantObject[]

# List Rooms
GET /api/rooms
  Query:
    type: "public" | "private" | "all"
    status: "active" | "ended" | "scheduled"
    limit: number
    offset: number
  Response:
    rooms: RoomObject[]
    total: number

# Update Room
PATCH /api/rooms/:id
  Body: Partial<RoomObject>
  Response: RoomObject

# Delete Room
DELETE /api/rooms/:id
  Response: { message: "Room deleted" }

# End Meeting
POST /api/rooms/:id/end
  Response: { message: "Meeting ended" }

# Join Room
GET /api/rooms/:id/join
  Query: token (for private rooms)
  Response:
    room: RoomObject
    livekit_token: string
    participants: ParticipantObject[]

# Leave Room
POST /api/rooms/:id/leave
  Response: { message: "Left room" }

# Invite Participants
POST /api/rooms/:id/invite
  Body:
    emails: string[]
    message: string (optional)
  Response:
    sent: number
    failed: number
```

### Recording Management

```yaml
# Start Recording
POST /api/rooms/:id/recording/start
  Response:
    recording_id: string
    started_at: timestamp

# Stop Recording
POST /api/rooms/:id/recording/stop
  Response:
    recording_id: string
    status: "completed"
    duration: number
    file_url: string

# List Recordings
GET /api/rooms/:id/recordings
  Response:
    recordings: RecordingObject[]
```

### Participant Management

```yaml
# List Participants
GET /api/rooms/:id/participants
  Response:
    participants: ParticipantObject[]

# Mute Participant
POST /api/rooms/:id/participants/:pid/mute
  Response: { status: "muted" }

# Unmute Participant
POST /api/rooms/:id/participants/:pid/unmute
  Response: { status: "unmuted" }

# Remove Participant
DELETE /api/rooms/:id/participants/:pid
  Response: { message: "Participant removed" }

# Transfer Host
POST /api/rooms/:id/transfer-host
  Body: { new_host_id: string }
  Response: { message: "Host transferred" }
```

## Database Schema

### rooms table

```sql
CREATE TABLE rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    host_id UUID NOT NULL REFERENCES users(id),
    type VARCHAR(20) NOT NULL DEFAULT 'public', -- 'public', 'private'
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- 'scheduled', 'active', 'ended'
    livekit_room_name VARCHAR(255) UNIQUE NOT NULL,
    max_participants INT DEFAULT 10,
    current_participants INT DEFAULT 0,
    settings JSONB DEFAULT '{}',
    started_at TIMESTAMP,
    ended_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_rooms_host ON rooms(host_id);
CREATE INDEX idx_rooms_status ON rooms(status);
CREATE INDEX idx_rooms_type ON rooms(type);
```

### participants table

```sql
CREATE TABLE participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    role VARCHAR(20) DEFAULT 'participant', -- 'host', 'participant'
    joined_at TIMESTAMP DEFAULT NOW(),
    left_at TIMESTAMP,
    is_removed BOOLEAN DEFAULT false,
    metadata JSONB DEFAULT '{}',
    CONSTRAINT unique_room_user UNIQUE (room_id, user_id)
);

CREATE INDEX idx_participants_room ON participants(room_id);
CREATE INDEX idx_participants_user ON participants(user_id);
```

### recordings table

```sql
CREATE TABLE recordings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    livekit_recording_id VARCHAR(255) UNIQUE,
    status VARCHAR(20) NOT NULL, -- 'recording', 'processing', 'completed', 'failed'
    file_url TEXT,
    file_size BIGINT,
    duration INT, -- seconds
    format VARCHAR(20), -- 'mp4', 'webm', 'mp3'
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_recordings_room ON recordings(room_id);
CREATE INDEX idx_recordings_status ON recordings(status);
```

### room_invitations table

```sql
CREATE TABLE room_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    inviter_id UUID NOT NULL REFERENCES users(id),
    invitee_id UUID REFERENCES users(id),
    invitee_email VARCHAR(255),
    token VARCHAR(255) UNIQUE,
    status VARCHAR(20) DEFAULT 'pending', -- 'pending', 'accepted', 'declined', 'expired'
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_invitations_room ON room_invitations(room_id);
CREATE INDEX idx_invitations_token ON room_invitations(token);
```

## LiveKit Integration

### Room Configuration

```go
// Create LiveKit room
func createLiveKitRoom(roomID string, settings RoomSettings) (*livekit.Room, error) {
    client := lksdk.NewRoomServiceClient(LIVEKIT_URL, LIVEKIT_API_KEY, LIVEKIT_API_SECRET)
    
    room, err := client.CreateRoom(context.Background(), &livekit.CreateRoomRequest{
        Name: roomID,
        EmptyTimeout: 60 * 10, // 10 minutes
        MaxParticipants: settings.MaxParticipants,
    })
    
    return room, err
}
```

### Generate Access Token

```go
func generateLiveKitToken(roomName, participantName, userID string, isHost bool) (string, error) {
    at := auth.NewAccessToken(LIVEKIT_API_KEY, LIVEKIT_API_SECRET)
    
    grant := &auth.VideoGrant{
        RoomJoin: true,
        Room:     roomName,
        CanPublish: true,
        CanSubscribe: true,
    }
    
    if isHost {
        grant.RoomAdmin = true
        grant.RoomRecord = true
    }
    
    at.AddGrant(grant).
        SetIdentity(userID).
        SetName(participantName).
        SetValidFor(2 * time.Hour)
    
    return at.ToJWT()
}
```

### WebHook Handling

```go
// Handle LiveKit webhooks
func handleLiveKitWebhook(w http.ResponseWriter, r *http.Request) {
    event := &livekit.WebhookEvent{}
    
    // Verify webhook signature
    if !verifyWebhook(r) {
        http.Error(w, "Invalid signature", http.StatusUnauthorized)
        return
    }
    
    // Parse event
    json.NewDecoder(r.Body).Decode(event)
    
    switch event.Event {
    case "participant_joined":
        handleParticipantJoined(event)
    case "participant_left":
        handleParticipantLeft(event)
    case "recording_finished":
        handleRecordingFinished(event)
    case "room_finished":
        handleRoomFinished(event)
    }
}
```

## WebSocket Events

### Server → Client Events

```typescript
// Participant joined
{
  type: "participant_joined",
  data: {
    user_id: string,
    name: string,
    avatar: string,
    role: "host" | "participant"
  }
}

// Participant left
{
  type: "participant_left",
  data: {
    user_id: string,
    reason: "left" | "removed" | "disconnected"
  }
}

// Meeting ended
{
  type: "meeting_ended",
  data: {
    room_id: string,
    ended_by: string,
    duration: number
  }
}

// Recording status changed
{
  type: "recording_status",
  data: {
    recording_id: string,
    status: "started" | "stopped" | "completed"
  }
}

// Settings updated
{
  type: "settings_updated",
  data: {
    settings: RoomSettings
  }
}

// Host transferred
{
  type: "host_transferred",
  data: {
    new_host_id: string,
    new_host_name: string
  }
}
```

## Error Handling

### Common Errors

| Error Code | Description | Resolution |
|------------|-------------|------------|
| `ROOM_NOT_FOUND` | Room doesn't exist | Verify room ID |
| `ROOM_FULL` | Max participants reached | Wait or create new room |
| `ROOM_ENDED` | Meeting already ended | Cannot rejoin |
| `ACCESS_DENIED` | Not invited/authorized | Request invitation |
| `INVALID_TOKEN` | LiveKit token expired | Request new token |
| `RECORDING_FAILED` | Recording error | Check LiveKit logs |
| `HOST_REQUIRED` | Operation requires host role | Contact host |

## Security Considerations

### Room Access Control
- ✅ Validate user permissions before join
- ✅ Check invitation tokens for private rooms
- ✅ Rate limit room creation
- ✅ Validate max participants limit

### LiveKit Security
- ✅ Short-lived access tokens (2 hours)
- ✅ Token includes user identity
- ✅ Webhook signature verification
- ✅ Secure API key storage

### Data Privacy
- ✅ Encrypt recordings at rest
- ✅ Auto-delete old recordings (30 days)
- ✅ User consent for recording
- ✅ GDPR-compliant data handling

## Testing Scenarios

- [ ] Create and join public room
- [ ] Create and join private room with invite
- [ ] Host can end meeting
- [ ] Participant can leave meeting
- [ ] Recording start/stop
- [ ] Screen sharing
- [ ] Participant removal
- [ ] Host transfer
- [ ] Connection recovery
- [ ] Multiple participants (5-10)
