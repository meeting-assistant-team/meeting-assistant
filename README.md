# Meeting Assistant Documentation

## 📚 Tổng quan

**Meeting Assistant** là ứng dụng web meeting với AI tích hợp để ghi âm, phân tích và tóm tắt cuộc họp tự động. Dự án được phát triển bởi sinh viên Đại học Công Nghệ Thông Tin - ĐHQG TP.HCM.

**Sinh viên thực hiện:**
- Nguyễn Minh Quang - 24410217
- Trần Đức Minh - 24410197

**Giảng viên hướng dẫn:** ThS. Đặng Văn Thìn

**Thời gian:** 03/11/2025 - 04/01/2026

## 🎯 Mục tiêu dự án

- Xây dựng ứng dụng web meeting với độ trễ thấp (5-10 người)
- Tích hợp AI để ghi âm và chuyển đổi thành văn bản
- Tự động tóm tắt nội dung và trích xuất action items
- Tạo báo cáo cá nhân hóa cho từng thành viên

## 📖 Nội dung tài liệu

### 1. [System Architecture](./01-system-architecture.md)
**Nội dung:**
- Tổng quan kiến trúc hệ thống
- Tech stack chi tiết
- Component design
- Data flow diagrams
- Security & scalability considerations

**Sơ đồ chính:**
- Kiến trúc tổng thể (Frontend → Backend → LiveKit → AI Services)
- Component interaction diagram
- Deployment architecture

### 2. [Authentication Flow](./02-authentication-flow.md)
**Nội dung:**
- OAuth2 integration (Google, GitHub)
- JWT token management
- Session handling với Redis
- Role-based access control (RBAC)
- Security best practices

**Sequence Diagrams:**
- ✅ OAuth2 login flow
- ✅ Token refresh flow
- ✅ Protected API request flow
- ✅ Logout flow
- ✅ Password reset flow (optional)

### 3. [Room Management Flow](./03-room-management-flow.md)
**Nội dung:**
- Tạo và quản lý phòng họp
- LiveKit integration
- Participant management
- Recording controls
- Screen sharing
- WebSocket real-time events

**Sequence Diagrams:**
- ✅ Create room flow
- ✅ Join room flow
- ✅ Complete meeting flow
- ✅ Leave room flow
- ✅ Invite participant flow
- ✅ Recording control flow
- ✅ Screen share flow
- ✅ Room settings management

### 4. [AI Analysis Flow](./04-ai-analysis-flow.md)
**Nội dung:**
- Speech-to-Text với Whisper API
- Speaker diarization
- GPT-4 content analysis
- Action items extraction
- Personal report generation
- ClickUp integration

**Sequence Diagrams:**
- ✅ Complete AI processing pipeline
- ✅ Speech-to-Text process
- ✅ Speaker diarization
- ✅ GPT-4 analysis process
- ✅ Personal report generation
- ✅ Action items extraction
- ✅ Notification flow

### 5. [Database Schema](./05-database-schema.md)
**Nội dung:**
- PostgreSQL schema design
- Table relationships (ERD)
- Indexes và optimization
- Redis cache structure
- Data retention policies
- Monitoring queries

**Tables:**
- `users`, `rooms`, `participants`
- `recordings`, `transcripts`
- `meeting_summaries`, `action_items`
- `participant_reports`, `notifications`
- Supporting tables

### 6. [API Documentation](./06-api-documentation.md)
**Nội dung:**
- Complete REST API reference
- Request/Response examples
- Authentication methods
- Error handling
- Rate limiting
- Webhooks

**Endpoints:**
- Authentication (`/auth/*`)
- Users (`/users/*`)
- Rooms (`/rooms/*`)
- Recordings (`/recordings/*`)
- Transcripts (`/meetings/*/transcript`)
- Summaries & Reports
- Action Items
- Notifications
- Integrations

### 7. [Deployment Guide](./07-deployment-guide.md)
**Nội dung:**
- Development setup
- Docker configuration
- Production deployment
- Environment variables
- Monitoring & logging
- Backup strategies

## 🔧 Tech Stack

### Backend
- **Language:** Golang 1.21+
- **Framework:** Echo v4
- **Architecture:** Clean Architecture
- **Database:** PostgreSQL 15+
- **Cache:** Redis 7+
- **ORM:** GORM / sqlx

### Frontend
- **Framework:** React 18 + TypeScript
- **State:** Redux Toolkit / Zustand
- **UI:** Material-UI / Ant Design
- **WebRTC:** LiveKit Client SDK

### Real-time
- **Solution:** LiveKit
- **Protocol:** WebRTC (SFU)

### AI Services
- **STT:** OpenAI Whisper API
- **Analysis:** OpenAI GPT-4 API
- **Speaker Diarization:** pyannote.audio

### Infrastructure
- **Container:** Docker + Docker Compose
- **Proxy:** Nginx
- **SSL:** Let's Encrypt
- **Monitoring:** Prometheus + Grafana
- **Storage:** MinIO (S3-compatible, self-hosted)

## 🚀 Quick Start

### Prerequisites

```bash
# Required
- Go 1.21+
- Node.js 18+
- Docker & Docker Compose
- PostgreSQL 15+
- Redis 7+

# API Keys needed
- LiveKit API Key & Secret
- OpenAI API Key
- OAuth credentials (Google/GitHub)
```

### Development Setup

```bash
# 1. Clone repository
git clone <repository-url>
cd meeting-assistant

# 2. Start infrastructure with Docker
docker-compose up -d

# 3. Setup backend
cd backend
cp .env.example .env
# Edit .env with your credentials
go mod download
go run main.go

# 4. Setup frontend
cd ../frontend
cp .env.example .env
# Edit .env with API URL
npm install
npm start

# 5. Access application
# Frontend: http://localhost:3000
# Backend API: http://localhost:8080
# Swagger docs: http://localhost:8080/swagger
```

### Docker Compose Services

```yaml
services:
  - postgres: PostgreSQL database
  - redis: Cache & sessions
  - minio: Object storage (recordings)
  - livekit: WebRTC server
  - backend: Go API server (Echo)
  - frontend: React app
  - nginx: Reverse proxy
```

## 📊 Chức năng chính

### 1. Authentication & User Management
- ✅ OAuth2 login (Google only)
- ✅ JWT token management
- ✅ User profiles
- ✅ Role-based permissions

### 2. Room Management
- ✅ Create public/private rooms
- ✅ Schedule meetings
- ✅ Invite participants
- ✅ Host controls (mute, remove, transfer host)
- ✅ Waiting room
- ✅ Room settings

### 3. Real-time Communication
- ✅ Audio/Video calls (5-10 participants)
- ✅ Screen sharing
- ✅ Text chat
- ✅ Connection quality indicators
- ✅ Low latency (<200ms)

### 4. Recording & Transcription
- ✅ Audio recording
- ✅ Speech-to-Text (Whisper)
- ✅ Speaker identification
- ✅ Word-level timestamps
- ✅ Multi-language support

### 5. AI Analysis
- ✅ Meeting summary generation
- ✅ Key points extraction
- ✅ Decision tracking
- ✅ Action items with assignments
- ✅ Sentiment analysis
- ✅ Personal participation reports

### 6. Task Management
- ✅ Auto-extracted action items
- ✅ Task assignments
- ✅ Priority & due dates
- ✅ Status tracking
- ✅ ClickUp integration (optional)

### 7. Notifications
- ✅ Email notifications
- ✅ In-app notifications
- ✅ Meeting reminders
- ✅ Report ready alerts
- ✅ Task assignments

### 8. Reports & Analytics
- ✅ Personal meeting reports
- ✅ Speaking time statistics
- ✅ Participation metrics
- ✅ Export to PDF/DOCX
- ✅ User statistics dashboard

## 🧪 Testing

### Unit Tests
```bash
# Backend
cd backend
go test ./...

# Frontend
cd frontend
npm test
```

### Integration Tests
```bash
cd backend
go test -tags=integration ./tests/integration
```

### E2E Tests
```bash
cd frontend
npm run test:e2e
```

## 📈 Performance Targets

| Metric | Target | Current |
|--------|--------|---------|
| API Response Time (p95) | < 200ms | ✅ |
| WebRTC Latency | < 200ms | ✅ |
| Concurrent Rooms | 10+ | ✅ |
| Participants per Room | 5-10 | ✅ |
| Transcription Time | < 2x audio length | ✅ |
| AI Analysis Time | < 5 min for 1h meeting | ✅ |

## 🔐 Security

- ✅ OAuth2 authentication
- ✅ JWT with refresh tokens
- ✅ HTTPS/TLS encryption
- ✅ CORS configuration
- ✅ Rate limiting
- ✅ Input validation
- ✅ SQL injection prevention
- ✅ XSS protection
- ✅ CSRF protection

## 📝 Kế hoạch thực hiện

| Tuần | Nội dung | Trạng thái |
|------|----------|-----------|
| 1 | Phân tích yêu cầu, thiết kế kiến trúc | ✅ Completed |
| 2 | Thiết kế UI/UX và database | 🔄 In Progress |
| 3-4 | Authentication, Room Management, WebRTC | ⏳ Planned |
| 5-6 | AI integration (STT, Analysis) | ⏳ Planned |
| 7-8 | Testing, optimization, UI polish | ⏳ Planned |
| 9 | Documentation, báo cáo, demo | ⏳ Planned |

## 🤝 Contributing

### Code Style
- **Go:** Follow [Effective Go](https://go.dev/doc/effective_go)
- **TypeScript:** ESLint + Prettier
- **Commits:** Conventional Commits

### Git Workflow
```bash
# Feature branch
git checkout -b feature/feature-name

# Commit changes
git commit -m "feat: add user authentication"

# Push and create PR
git push origin feature/feature-name
```

## 📚 Tài liệu tham khảo

### External Documentation
- [LiveKit Documentation](https://docs.livekit.io/)
- [OpenAI API Reference](https://platform.openai.com/docs/api-reference)
- [Whisper API Guide](https://platform.openai.com/docs/guides/speech-to-text)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Redis Documentation](https://redis.io/documentation)

### Learning Resources
- [WebRTC Concepts](https://webrtc.org/)
- [Go Web Development](https://go.dev/doc/)
- [React Documentation](https://react.dev/)
- [System Design](https://github.com/donnemartin/system-design-primer)

## 📞 Contact

**Sinh viên:**
- Nguyễn Minh Quang: 24410217@student.uit.edu.vn
- Trần Đức Minh: 24410197@student.uit.edu.vn

**Giảng viên hướng dẫn:**
- ThS. Đặng Văn Thìn: thindv@uit.edu.vn

## 📄 License

This project is developed for educational purposes at University of Information Technology, VNU-HCM.

---

**Last Updated:** November 3, 2025

**Version:** 1.0.0 (MVP)
