# Luồng Xử Lý Ghi Âm đến AssemblyAI

## 📊 Tổng Quan Kiến Trúc

**Luồng xử lý tổng quát:**

1. **Phòng Kết Thúc** → LiveKit gửi webhook `room_finished` → Cập nhật trạng thái phòng
2. **Ghi Âm Sẵn Sàng** → LiveKit gửi webhook `egress_ended` → Trích xuất URL ghi âm
3. **Gửi đến AI** → Gọi `AIService.SubmitToAssemblyAI(roomID, recordingURL)` 
4. **Xử Lý AssemblyAI** → Chuyển đổi văn bản + Phân tách người nói (~23s cho audio 30 phút)
5. **Webhook Callback** → AssemblyAI gửi bản ghi âm qua webhook
6. **Lưu Trữ Bản Ghi** → Tạo Transcript trong DB với nhãn người nói
7. **Sẵn Sàng cho Giai Đoạn 2** → Phân tích AI với Groq LLM

## 🎬 Cấu Hình LiveKit Recording

**Phương thức:** Legacy Recording thông qua Dashboard config (đơn giản, ổn định)

**Lưu ý quan trọng:** Legacy recording vẫn sử dụng webhook `egress_ended`, chỉ khác ở cách config (Dashboard vs code)

**Điều kiện tiên quyết:**
- Phòng LiveKit đã được tạo
- S3 storage được cấu hình **MỘT LẦN** trong LiveKit Dashboard (Settings → Recording)
- Auto-record enabled trong Dashboard
- Webhooks được bật trong LiveKit Dashboard

**Các sự kiện webhook bắt buộc:**
- `room_finished` - Người tham gia cuối cùng rời khỏi phòng
- `egress_ended` ⭐ **QUAN TRỌNG** - Ghi âm sẵn sàng để tải xuống (dùng cho cả legacy và modern egress)

**Dữ liệu chính từ webhook egress_ended:**
- `room.name` - Định danh phòng LiveKit
- `egress_info.file_results[0].location` - URL HTTP để tải xuống ghi âm
- `egress_info.egress_id` - ID của egress job
- `egress_info.file_results[0].size` - Kích thước file (bytes)
- `egress_info.file_results[0].duration` - Thời lượng (milliseconds)

## 🔧 Các Thành Phần Kiến Trúc

**WebhookHandler** (internal/adapter/handler/webhook.go)
- Nhận các sự kiện webhook từ LiveKit
- Định tuyến dựa trên loại sự kiện
- `handleRecordingFinished()` - Trích xuất URL ghi âm, gọi AIService

**AIService** (internal/usecase/ai/service.go)
- `SubmitToAssemblyAI(meetingID, recordingURL)` - Gửi đến AssemblyAI với cơ chế retry
- `HandleAssemblyAIWebhook()` - Xử lý callback từ AssemblyAI, lưu trữ bản ghi

**Repositories:**
- `AIJobRepository` - Theo dõi các công việc chuyển đổi văn bản (pending, submitted, completed, failed)
- `TranscriptRepository` - Lưu trữ bản ghi cuối cùng với thông tin người nói

## 📡 Cấu Hình Bắt Buộc

**Trong LiveKit Dashboard (chỉ cần cấu hình MỘT LẦN):**
1. Truy cập Settings → Recording
2. Chọn S3 Storage và nhập credentials:
   - S3 Endpoint (hoặc AWS region)
   - Access Key ID
   - Secret Access Key
   - Bucket Name
3. Enable "Auto-record rooms"
4. Save settings

**Biến môi trường backend:**
```
ASSEMBLYAI_API_KEY=aai_xxxxx                          # Lấy từ AssemblyAI dashboard
ASSEMBLYAI_WEBHOOK_SECRET=wh_xxxxx                    # Tạo secret cho webhook signature
ASSEMBLYAI_WEBHOOK_BASE_URL=https://your-domain/v1/webhooks/assemblyai

LIVEKIT_URL=wss://your-livekit.com
LIVEKIT_API_KEY=APIxxxxx
LIVEKIT_API_SECRET=secretxxxxx
```

Cho môi trường dev local với ngrok: Sử dụng URL ngrok làm ASSEMBLYAI_WEBHOOK_BASE_URL

## 🔄 Chi Tiết Pipeline Xử Lý

### Giai Đoạn 1: Trích Xuất Ghi Âm
- LiveKit phát hiện ghi âm hoàn tất → gửi webhook `egress_ended`
- Backend trích xuất URL từ `egress_info.file_results[0].location`
- Tìm phòng theo tên phòng LiveKit
- Gọi AIService để gửi ghi âm

### Giai Đoạn 2: Gửi đến AssemblyAI
- Tạo AIJob với trạng thái: `pending`
- Gửi URL ghi âm đến AssemblyAI API
- Bao gồm URL webhook cho callback
- Nhận về external_job_id (transcript ID)
- Cập nhật trạng thái AIJob: `submitted`
- Cơ chế retry: Exponential backoff (1s, 2s, 4s, 8s, 15s) - tối đa 3 lần thử

### Giai Đoạn 3: Xử Lý AssemblyAI
- AssemblyAI xử lý audio không đồng bộ
- Trích xuất:
  - **Văn bản đầy đủ** với timestamp cấp từ
  - **Phân tách người nói** (tự động nhận diện các người nói khác nhau)
  - **Phát hiện ngôn ngữ**
  - **Điểm tin cậy** cho mỗi từ

### Giai Đoạn 4: Webhook Callback
- AssemblyAI gửi webhook khi xử lý hoàn tất
- Bao gồm bản ghi đầy đủ với thông tin người nói
- Xác thực chữ ký (HMAC-SHA256)

### Giai Đoạn 5: Lưu Trữ Bản Ghi
- Xác minh chữ ký webhook với ASSEMBLYAI_WEBHOOK_SECRET
- Phân tích phản hồi từ AssemblyAI
- Tạo bản ghi Transcript với:
  - Văn bản đầy đủ
  - Ngôn ngữ phát hiện
  - Số lượng người nói
  - Phản hồi JSON thô (lưu dưới dạng JSONB)
- Cập nhật AIJob: status → `completed`, liên kết đến transcript_id

## 🗄️ Các Bảng Database Được Sử Dụng

**ai_jobs**
- Theo dõi vòng đời công việc chuyển đổi văn bản
- Các trường: id, meeting_id, job_type, status, external_job_id, recording_url, transcript_id
- Trạng thái: pending → submitted → completed (hoặc failed)

**transcripts**
- Lưu trữ bản ghi cuối cùng
- Các trường: id, recording_id, meeting_id, text, language, speaker_count, raw_data
- Raw data chứa phản hồi đầy đủ từ AssemblyAI cho xử lý Giai đoạn 2

## ⚡ Các Quyết Định Thiết Kế Chính

1. **Legacy Recording thay vì Egress** - Đơn giản hơn, ít lỗi hơn, chỉ 1 webhook event
2. **S3 config trong Dashboard** - Bảo mật hơn, không hardcode credentials trong code
3. **Webhook-driven** - Không polling, hoàn toàn bất đồng bộ
4. **Speaker diarization tích hợp sẵn** - AssemblyAI xử lý tự động, không cần API call thêm
5. **Exponential backoff** - Xử lý lỗi tạm thời một cách uyển chuyển
6. **Xác thực chữ ký** - Xác nhận webhook từ AssemblyAI là chính thức
7. **Lưu trữ JSONB** - Phản hồi đầy đủ từ AssemblyAI sẵn sàng cho các cải tiến tương lai

## 🚨 Xử Lý Lỗi

**Thiếu URL ghi âm** → Ghi log cảnh báo, trả về 200 OK, có thể retry sau

**Gửi AssemblyAI thất bại** → Exponential backoff retry, sau đó đánh dấu failed với thông báo lỗi

**Chữ ký webhook không hợp lệ** → Từ chối request (400), ghi log cảnh báo bảo mật

**Timeout xử lý** (task Giai đoạn 2) → Triển khai polling hoặc khôi phục timeout

## 📚 Tài Liệu Liên Quan

- [Tài liệu LiveKit Recording](https://docs.livekit.io/realtime/server/recording/)
- [Tài liệu LiveKit Webhooks](https://docs.livekit.io/realtime/server/webhooks/)
- [AssemblyAI Transcription API](https://www.assemblyai.com/docs/transcription)
- [AssemblyAI Speaker Diarization](https://www.assemblyai.com/docs/models/speaker-diarization)

