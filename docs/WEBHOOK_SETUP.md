# LiveKit Webhook Setup Guide

## 📡 Tổng quan

Webhook cho phép LiveKit tự động thông báo cho backend khi có sự kiện xảy ra:
- **participant_joined** - User join room
- **participant_left** - User rời room hoặc disconnect (tắt browser)
- **room_started** - Room bắt đầu
- **room_finished** - Room kết thúc (tất cả users đã rời)

## ✅ Đã implement

### 1. Webhook Handler

File: `/internal/adapter/handler/webhook.go`

```go
type WebhookHandler struct {
    roomService   roomUsecase.Service
    webhookSecret string
}
```

**Endpoints:**
- `POST /v1/webhooks/livekit` - Nhận webhook từ LiveKit

**Auto-processing:**
- `participant_left` → Tự động gọi `LeaveRoom()`
- `room_finished` → Tự động gọi `EndRoom()`

### 2. Service Methods

Đã thêm vào `/internal/usecase/room/service.go`:

```go
// Tìm room theo LiveKit room name
GetRoomByLivekitName(ctx context.Context, livekitName string) (*entities.Room, error)

// Cập nhật trạng thái participant
UpdateParticipantStatus(ctx context.Context, roomID, userID uuid.UUID, status string) error
```

### 3. Router Integration

- Webhook route đã được đăng ký tại: `POST /v1/webhooks/livekit`
- **Không yêu cầu auth** (LiveKit sẽ gọi trực tiếp)
- WebhookSecret được load từ env: `LIVEKIT_WEBHOOK_SECRET`

## 🔧 Cấu hình

### Local Development (với ngrok)

**Bước 1: Cài ngrok**
```bash
# macOS
brew install ngrok

# hoặc download từ https://ngrok.com/download
```

**Bước 2: Expose local server**
```bash
# Giả sử server đang chạy ở port 8080
ngrok http 8080
```

Output sẽ hiển thị URL:
```
Forwarding  https://abc123.ngrok.io -> http://localhost:8080
```

**Bước 3: Config LiveKit Cloud Dashboard**

1. Vào: https://cloud.livekit.io/
2. Chọn project: `meeting-assistant-39o34tzz`
3. Menu bên trái: **Settings** → **Webhooks**
4. Click **Add Webhook**
5. Điền thông tin:
   ```
   Webhook URL: https://abc123.ngrok.io/v1/webhooks/livekit
   
   Events to send:
   ✅ participant_joined
   ✅ participant_left  ⭐ (Important)
   ✅ room_started
   ✅ room_finished     ⭐ (Important)
   
   Secret (optional): [để trống cho dev, hoặc dùng random string]
   ```
6. Click **Save**

**Bước 4: Cập nhật .env (nếu có secret)**

```bash
# File: .env
LIVEKIT_WEBHOOK_SECRET=your-webhook-secret-here
```

### Production Deployment

**Option A: Deploy trên VPS/Cloud**

Nếu backend deploy tại `https://api-meeting.infoquang.id.vn`:

```
Webhook URL: https://api-meeting.infoquang.id.vn/v1/webhooks/livekit
```

**Option B: Sử dụng Cloudflare Tunnel (thay ngrok)**

```bash
# Cài cloudflared
brew install cloudflare/cloudflare/cloudflared

# Run tunnel
cloudflared tunnel --url http://localhost:8080
```

## 🧪 Testing

### Test 1: Tắt browser (không click Leave)

**Scenario:**
1. User join room từ FE
2. Tắt browser/tab (không gọi API leave)
3. Sau ~5 giây, LiveKit phát hiện disconnect

**Expected behavior:**
```
Backend logs:
📡 LiveKit Webhook Event: participant_left
👋 Participant left: <user-id> from room <room-name>
✅ Auto-left user <user-id> from room <room-id>
```

**Database check:**
```sql
SELECT * FROM participants 
WHERE room_id = '<room-id>' AND user_id = '<user-id>';
-- left_at sẽ không null
```

### Test 2: Tất cả users tắt browser

**Scenario:**
1. Nhiều users join room
2. Tất cả tắt browser cùng lúc

**Expected behavior:**
```
Backend logs:
📡 LiveKit Webhook Event: participant_left (user 1)
📡 LiveKit Webhook Event: participant_left (user 2)
📡 LiveKit Webhook Event: participant_left (user 3)
...
📡 LiveKit Webhook Event: room_finished
🏁 Room finished: <room-name>
✅ Auto-ended room <room-id>
```

**Database check:**
```sql
SELECT status, ended_at, current_participants 
FROM rooms 
WHERE id = '<room-id>';

-- status = 'ended'
-- ended_at = timestamp
-- current_participants = 0
```

### Test 3: Manual curl test

```bash
# Giả lập webhook từ LiveKit
curl -X POST http://localhost:8080/v1/webhooks/livekit \
  -H "Content-Type: application/json" \
  -d '{
    "event": "participant_left",
    "room": {
      "name": "test-room-abc123"
    },
    "participant": {
      "identity": "550e8400-e29b-41d4-a716-446655440000"
    },
    "createdAt": 1700000000
  }'
```

**Expected response:**
```json
{
  "status": "ok",
  "event": "participant_left"
}
```

## 📊 Monitoring

### Check webhook delivery trong LiveKit Dashboard

1. Vào: https://cloud.livekit.io/
2. Settings → Webhooks
3. Click vào webhook đã tạo
4. Xem **Recent Deliveries**:
   - Status code (200 = success)
   - Response time
   - Payload đã gửi

### Backend logs

Khi webhook hoạt động, bạn sẽ thấy:

```
📡 LiveKit Webhook Event: participant_joined
👤 Participant joined: <user-id> in room <room-name>

📡 LiveKit Webhook Event: participant_left
👋 Participant left: <user-id> from room <room-name>
✅ Auto-left user <user-id> from room <room-id>

📡 LiveKit Webhook Event: room_finished
🏁 Room finished: <room-name>
✅ Auto-ended room <room-id>
```

## 🔒 Security (Production)

### 1. Webhook Secret Validation

Hiện tại code **không verify** webhook signature (để đơn giản).

Nếu muốn verify signature trong production:

```go
// webhook.go
import "github.com/livekit/protocol/webhook"

func (h *WebhookHandler) HandleLiveKitWebhook(c echo.Context) error {
    // Get raw body
    body, _ := io.ReadAll(c.Request().Body)
    
    // Verify signature
    authHeader := c.Request().Header.Get("Authorization")
    receiver := webhook.NewReceiver(h.webhookSecret)
    
    event, err := receiver.Receive(body, authHeader)
    if err != nil {
        return c.JSON(400, map[string]string{"error": "invalid signature"})
    }
    
    // Process event...
}
```

### 2. IP Whitelist (Optional)

Chỉ cho phép webhook từ IP của LiveKit:

```go
// middleware
func LiveKitWebhookMiddleware() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            ip := c.RealIP()
            // Check if IP is from LiveKit
            // ...
            return next(c)
        }
    }
}
```

## 🎯 Next Steps

1. **Deploy backend** lên server có public URL
2. **Config webhook** trong LiveKit Cloud Dashboard
3. **Test workflow:**
   - Join room → Check `participant_joined` webhook
   - Close browser → Check `participant_left` webhook
   - Room auto-end → Check `room_finished` webhook
4. **Monitor logs** để ensure webhooks hoạt động đúng
5. **Implement signature verification** cho production (optional nhưng recommended)

## ❓ Troubleshooting

### Webhook không được gọi

**Check:**
1. Backend có public URL chưa? (ngrok/cloudflare tunnel/deployed)
2. URL trong LiveKit Dashboard đúng chưa?
3. Backend server đang chạy?
4. Firewall có block incoming requests không?

**Test manual:**
```bash
curl -X POST <your-webhook-url> -H "Content-Type: application/json" -d '{"event":"test"}'
```

### Webhook returns 404

- Check route đã đăng ký đúng: `POST /v1/webhooks/livekit`
- Check server logs có nhận request không

### Participant không auto-leave

**Possible causes:**
1. LiveKit chưa phát hiện disconnect (chờ ~5 giây)
2. Webhook event `participant_left` chưa được enable
3. Handler có lỗi (check logs)

**Debug:**
```bash
# Check participant records
psql -U postgres -d meeting_assistant -c "
SELECT p.*, u.email 
FROM participants p 
JOIN users u ON p.user_id = u.id 
WHERE p.room_id = '<room-id>';
"
```

## 📚 References

- [LiveKit Webhooks Documentation](https://docs.livekit.io/realtime/server/webhooks/)
- [LiveKit Cloud Dashboard](https://cloud.livekit.io/)
- [ngrok Documentation](https://ngrok.com/docs)
