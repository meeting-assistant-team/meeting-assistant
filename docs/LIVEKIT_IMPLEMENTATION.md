# LiveKit Integration - Implementation Summary

## ✅ Hoàn thành

### 1. LiveKit Client Infrastructure
**File:** `internal/infrastructure/external/livekit/client.go`

- ✅ Interface `Client` với methods: CreateRoom, DeleteRoom, GenerateToken, ListParticipants
- ✅ Real client implementation sử dụng `livekit/server-sdk-go/v2`
- ✅ Mock client implementation để test không cần server thật
- ✅ Token generation với `livekit/protocol/auth`
- ✅ Configurable qua env `LIVEKIT_USE_MOCK`

**Features:**
- CreateRoom với options (MaxParticipants, EmptyTimeout, Metadata)
- DeleteRoom với cleanup
- GenerateToken với granular permissions (CanPublish, CanSubscribe, RoomAdmin...)
- ListParticipants với participant info

### 2. Configuration
**File:** `pkg/config/config.go`

Added `LiveKitConfig` struct:
```go
type LiveKitConfig struct {
    URL       string // LiveKit server URL
    APIKey    string // API key
    APISecret string // API secret
    UseMock   bool   // Mock mode toggle
}
```

Environment variables:
- `LIVEKIT_URL` (default: `ws://localhost:7880`)
- `LIVEKIT_API_KEY` (default: `devkey`)
- `LIVEKIT_API_SECRET` (default: `secret`)
- `LIVEKIT_USE_MOCK` (default: `true`)

### 3. Room Service Updates
**File:** `internal/usecase/room/room_service.go`

- ✅ Inject `livekit.Client` vào RoomService
- ✅ CreateRoom gọi LiveKit API trước khi save DB
- ✅ CreateRoom trả về `CreateRoomOutput` với token
- ✅ Cleanup logic: delete LiveKit room nếu DB fails
- ✅ GenerateParticipantToken method
- ✅ GetLivekitURL method

**New Output Struct:**
```go
type CreateRoomOutput struct {
    Room          *entities.Room
    LivekitToken  string
    LivekitURL    string
    LivekitRoomID string
}
```

### 4. Handler Updates
**File:** `internal/adapter/handler/room.go`

- ✅ CreateRoom handler trả về `CreateRoomResponse` với token
- ✅ JoinRoom handler generate token thật (không còn dummy)
- ✅ Error handling cho LiveKit failures

### 5. DTO Updates
**File:** `internal/adapter/dto/room/response.go`

Added `CreateRoomResponse`:
```go
type CreateRoomResponse struct {
    Room         *RoomResponse
    LivekitToken string
    LivekitURL   string
}
```

### 6. Main Wiring
**File:** `cmd/api/main.go`

- ✅ Initialize LiveKit client với config
- ✅ Log mode (MOCK vs Real)
- ✅ Inject vào RoomService

### 7. Testing Infrastructure
**Files:**
- `.env.example` - Updated with LiveKit config
- `scripts/test_room_api.sh` - Full API test flow
- `scripts/quick_test.sh` - Quick test with token
- `TESTING_ROOM_API.md` - Comprehensive testing guide

## 🎯 Capabilities

### Mock Mode (LIVEKIT_USE_MOCK=true)
✅ Test backend **mà không cần LiveKit server**  
✅ Generate real JWT tokens (có thể decode)  
✅ Simulate room creation success  
✅ Không call external API  

**Use case:** Development, unit tests, CI/CD without LiveKit dependency

### Real Mode (LIVEKIT_USE_MOCK=false)
✅ Connect tới LiveKit server thật  
✅ Create rooms trong LiveKit  
✅ Generate tokens với real room context  
✅ List participants từ LiveKit  

**Use case:** Production, integration tests, end-to-end testing

## 📊 API Response Examples

### POST /rooms (Create Room)

**Request:**
```json
{
  "name": "Product Planning",
  "type": "public",
  "max_participants": 10
}
```

**Response:**
```json
{
  "room": {
    "id": "uuid",
    "name": "Product Planning",
    "livekit_room_name": "room-xxxxx",
    "status": "scheduled",
    ...
  },
  "livekit_token": "eyJhbGci...jwt-token",
  "livekit_url": "ws://localhost:7880"
}
```

### POST /rooms/:id/join (Join Room)

**Response:**
```json
{
  "room": { ... },
  "livekit_token": "eyJhbGci...participant-token",
  "livekit_url": "ws://localhost:7880",
  "participants": [...],
  "participant": { ... }
}
```

## 🔐 Token Structure

Tokens include:
- `video.room`: room name
- `video.roomJoin`: true
- `video.canPublish`: true (for participants)
- `video.canSubscribe`: true
- `video.roomAdmin`: true (host only)
- `sub`: user UUID
- `name`: "Host" or "Participant"
- `exp`: 24 hours from now

## 🧪 How to Test

### Quick Test (Mock Mode)

1. Ensure `.env` has `LIVEKIT_USE_MOCK=true`
2. Start backend: `go run cmd/api/main.go`
3. Login via OAuth to get token
4. Run: `./scripts/quick_test.sh <your-token>`

### Full Test Flow

1. Run: `./scripts/test_room_api.sh`
2. Follow prompts to login and test all endpoints

### Manual cURL

See `TESTING_ROOM_API.md` for detailed manual testing steps.

## 🚀 Production Deployment

1. Set up LiveKit server (self-hosted or cloud)
2. Update `.env`:
   ```env
   LIVEKIT_URL=wss://your-livekit-server.com
   LIVEKIT_API_KEY=your-production-key
   LIVEKIT_API_SECRET=your-production-secret
   LIVEKIT_USE_MOCK=false
   ```
3. Deploy backend
4. Frontend connects using returned `livekit_token` and `livekit_url`

## 📦 Dependencies Added

```
github.com/livekit/server-sdk-go/v2 v2.12.8
github.com/livekit/protocol v1.43.0
```

Plus transitive dependencies for WebRTC, protobuf, etc.

## ⚠️ Known Limitations / TODO

- [ ] DB transaction cho CreateRoom (currently sequential, not atomic)
- [ ] LiveKit webhook handlers (room events, participant events)
- [ ] Recording control (start/stop recording via LiveKit egress)
- [ ] Token refresh for long meetings (tokens expire in 24h)
- [ ] Room cleanup job (delete ended rooms from LiveKit)
- [ ] Metrics/monitoring for LiveKit API calls
- [ ] Retry logic for transient LiveKit failures

## 🎓 Key Design Decisions

1. **Mock mode by default** - Developers can test without external deps
2. **Real JWT tokens in mock** - Ensures token format compatibility
3. **Cleanup on failure** - Delete LiveKit room if DB insert fails
4. **Interface-based design** - Easy to swap implementations or add new providers
5. **Config-driven** - All settings via environment variables

## 🔗 Integration Points

### Backend → LiveKit
- Room creation: `livekitClient.CreateRoom()`
- Token generation: `livekitClient.GenerateToken()`
- Room deletion: `livekitClient.DeleteRoom()`

### Frontend → LiveKit (future)
```typescript
import { Room } from 'livekit-client';

const room = new Room();
await room.connect(livekitUrl, livekitToken);
```

### LiveKit → Backend (webhooks, future)
```go
// Handle webhook from LiveKit
POST /webhooks/livekit
{
  "event": "participant_joined",
  "room": "...",
  "participant": "..."
}
```

## 📝 Code Quality

- ✅ No compile errors
- ✅ Interface compliance verified
- ✅ Config validation
- ✅ Error handling with cleanup
- ✅ Logging for observability
- ✅ Comments and documentation

## 🎉 Result

**Backend hoàn toàn functional và có thể test ngay** với mock mode. Không cần LiveKit server để verify API hoạt động đúng. Token generation real và có thể decode, ready cho frontend integration.

**Next step:** Frontend tích hợp LiveKit client SDK và connect bằng token từ API.
