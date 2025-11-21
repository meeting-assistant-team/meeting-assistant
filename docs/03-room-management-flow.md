# Room Management & Meeting Flow

## Overview

Hệ thống quản lý phòng họp cho phép người dùng tạo, tham gia và quản lý các cuộc họp trực tuyến với audio/video real-time thông qua LiveKit.

## Room Access Control

### Join Flow
- **Tất cả participants đều phải được host approve**
- Click link → Login → Waiting room
- Host nhận notification → Approve/Deny
- Không có auto-admit, host kiểm soát hoàn toàn

### Exception: Host
- Host là người tạo room → tự động join ngay
- Không cần approval cho chính host

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
    
    B->>B: Generate shareable join link
    Note over B: /rooms/{room_id}/join
    
    B-->>F: { room, livekit_token, livekit_url, join_url }
    
    F->>F: Navigate to room page
    F->>F: Show "Copy Link" button with join_url
    F->>LK: Connect via WebSocket
    Note over F,LK: Using LiveKit SDK
    
    F-->>U: Show meeting room UI + shareable link
```

## Join Room Flow (Via Shared Link)

```mermaid
sequenceDiagram
    participant P as Participant
    participant F as Frontend
    participant B as Backend
    participant LK as LiveKit Server
    participant DB as Database
    participant WS as WebSocket (Notifications)
    participant H as Host
    
    Note over P: Host shared link:<br/>/rooms/{room_id}/join
    
    P->>F: Click on shared link
    F->>F: Extract room_id from URL
    
    alt User not logged in
        F->>F: Redirect to login page
        Note over F: Save join URL to redirect<br/>after successful login
        P->>F: Complete login/signup
        F->>F: Redirect back to join URL
    end
    
    Note over F,B: User is now authenticated
    
    F->>B: POST /api/rooms/:id/join
    Note over F,B: Authorization: Bearer {token}<br/>(Required for identity)
    
    B->>DB: SELECT room WHERE id = ?
    
    alt Room not found
        B-->>F: 404 Not Found
        F-->>P: "Room does not exist"
    else Room found
        B->>DB: Check room status
        alt Room ended
            B-->>F: 400 Bad Request "Room has ended"
            F-->>P: "This meeting has ended"
        else Room active/scheduled
            B->>DB: Check room capacity
            
            alt Room full
                B-->>F: 400 Bad Request "Room is full"
                F-->>P: "Room is at maximum capacity"
            else Room available
                B->>DB: Check if user is host
                
                alt User is host
                    B->>DB: UPDATE participants<br/>SET status = 'joined'
                    
                    B->>B: Generate LiveKit token
                    B->>DB: UPDATE rooms SET participant_count++
                    
                    B-->>F: { room, livekit_token, participants }
                    F->>LK: Connect to LiveKit room
                    F-->>P: Show meeting UI (as host)
                    
                else User is participant
                    B->>DB: INSERT INTO participants<br/>SET status = 'waiting'
                    
                    B->>WS: Notify host
                    Note over WS: { event: "join_request",<br/>user: {...} }
                    
                    WS-->>H: Show join request notification
                    
                    B-->>F: { status: "waiting_for_approval" }
                    F-->>P: Show "Waiting for host to let you in..."
                    
                    Note over P,H: Participant waits in waiting room
                    
                    alt Host approves
                        H->>F: Click "Admit"
                        F->>B: POST /api/rooms/:id/participants/:pid/admit
                        
                        B->>DB: UPDATE participants<br/>SET status = 'joined'
                        B->>B: Generate LiveKit token
                        B->>DB: UPDATE rooms SET participant_count++
                        
                        B->>WS: Notify participant
                        Note over WS: { event: "admission_approved" }
                        
                        WS-->>F: Approved notification
                        F->>LK: Connect to LiveKit room
                        F-->>P: Show meeting UI
                        
                    else Host rejects
                        H->>F: Click "Deny"
                        F->>B: POST /api/rooms/:id/participants/:pid/deny
                        
                        B->>DB: UPDATE participants<br/>SET status = 'denied'
                        
                        B->>WS: Notify participant
                        Note over WS: { event: "admission_denied" }
                        
                        WS-->>F: Denied notification
                        F-->>P: "Host denied your request"
                    end
                end
            end
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

## Share Room Link Flow

```mermaid
sequenceDiagram
    participant H as Host
    participant F as Frontend
    participant C as Clipboard/Share API
    participant P as Participants (External)
    
    Note over H,F: Host is in meeting room
    
    H->>F: Click "Share Link" or "Copy Link"
    
    alt Copy to clipboard
        F->>F: Get join_url from room data
        Note over F: /rooms/{room_id}/join
        F->>C: navigator.clipboard.writeText(url)
        F-->>H: Show "Link copied!" notification
        H->>H: Paste link in chat/email/etc
        
    else Use Web Share API (mobile)
        F->>C: navigator.share({ url, title })
        C->>C: Show native share dialog
        H->>C: Choose app (WhatsApp, Email, etc)
        C->>P: Send link via chosen app
    end
    
    Note over P: Participant receives link
    P->>P: Click on link
    Note over P: → Triggers "Join Room Flow"
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
    max_participants: number (default: 10)
    settings:
      enable_recording: boolean
      enable_chat: boolean
  Response:
    room: RoomObject
    livekit_token: string
    livekit_url: string
    join_url: string

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

# Join Room (via shared link)
POST /api/rooms/:id/join
  Headers: Authorization: Bearer {token}
  Response (if host):
    room: RoomObject
    livekit_token: string
    livekit_url: string
    participants: ParticipantObject[]
    participant: ParticipantObject (current user)
  Response (if participant - waiting):
    status: "waiting_for_approval"
    message: "Waiting for host to let you in"
    participant: ParticipantObject (status: 'waiting')

# Admit Participant (Host only)
POST /api/rooms/:id/participants/:pid/admit
  Response: { message: "Participant admitted" }

# Deny Participant (Host only)  
POST /api/rooms/:id/participants/:pid/deny
  Body: { reason: string }
  Response: { message: "Participant denied" }

# Get Waiting Participants (Host only)
GET /api/rooms/:id/participants/waiting
  Response:
    participants: ParticipantObject[] (status: 'waiting')

# Leave Room
POST /api/rooms/:id/leave
  Response: { message: "Left room" }
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
```

### participants table

```sql
CREATE TABLE participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    role VARCHAR(20) DEFAULT 'participant', -- 'host', 'participant'
    status VARCHAR(20) DEFAULT 'waiting', -- 'waiting', 'joined', 'left', 'denied'
    joined_at TIMESTAMP,
    left_at TIMESTAMP,
    is_removed BOOLEAN DEFAULT false,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT unique_room_user UNIQUE (room_id, user_id)
);

CREATE INDEX idx_participants_room ON participants(room_id);
CREATE INDEX idx_participants_user ON participants(user_id);
CREATE INDEX idx_participants_status ON participants(status);
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

### ~~room_invitations table~~ (REMOVED - Not needed with link sharing)

**Note:** With the new link-sharing approach, we don't need a separate invitations table. 
Anyone with the link can join (if room not full and not ended).

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

// Join request (waiting room)
{
  type: "join_request",
  data: {
    user_id: string,
    name: string,
    avatar: string,
    participant_id: string
  }
}

// Admission approved
{
  type: "admission_approved",
  data: {
    participant_id: string,
    livekit_token: string
  }
}

// Admission denied
{
  type: "admission_denied",
  data: {
    participant_id: string,
    reason: string
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
- ✅ **Require authentication** - Must be logged in to join any room
- ✅ **Identify users** - Track who joined via JWT token user_id
- ✅ **Host approval required** - All participants must be approved by host
- ✅ **Waiting room** - Participants wait for host approval (except host)
- ✅ **Host auto-join** - Host (creator) joins immediately without approval
- ✅ Check room status (not ended) before allowing join
- ✅ Rate limit room creation per user
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

- [ ] **Host creates room** → gets join link
- [ ] **Host joins own room** → auto-admitted (no waiting)
- [ ] **Participant joins via link** → enters waiting room
- [ ] **Host receives notification** → when someone requests to join
- [ ] **Host admits participant** → participant joins meeting
- [ ] **Host denies participant** → participant gets rejection message
- [ ] **Waiting room UI** → show list of waiting participants to host
- [ ] **Participant waiting UI** → show "Waiting for host..." message
- [ ] **Multiple waiting** → multiple participants in waiting room
- [ ] **Room full** → new joins rejected with error (after admission)
- [ ] **Room ended** → cannot join via link
- [ ] **Host can end meeting** → all participants disconnected
- [ ] **Participant can leave meeting** → removed from participant list
- [ ] **Copy link functionality** → test clipboard API
- [ ] **Mobile share** → test Web Share API on mobile devices
