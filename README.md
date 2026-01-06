# Meeting Assistant Documentation

## 📚 Tổng quan

**Meeting Assistant** là ứng dụng web meeting với AI tích hợp để ghi âm, phân tích và tóm tắt cuộc họp tự động. Dự án được phát triển bởi sinh viên Đại học Công Nghệ Thông Tin - ĐHQG TP.HCM.

**Sinh viên thực hiện:**
- Nguyễn Minh Quang - 24410217
- Trần Đức Minh - 24410197

**Giảng viên hướng dẫn:** ThS. Đặng Văn Thìn

**Thời gian thực hiện:** 03/11/2024 - 04/01/2025 (9 tuần)

## 🎯 Mục tiêu dự án

- Xây dựng ứng dụng web meeting với độ trễ thấp (hỗ trợ 5 người/phòng)
- Tích hợp AI để ghi âm và chuyển đổi giọng nói thành văn bản (Speech-to-Text)
- Tự động phân tích, tóm tắt nội dung và trích xuất action items từ cuộc họp
- Self-hosted infrastructure với chi phí tối ưu

## 📖 Nội dung tài liệu

### 1. [System Architecture](./01-system-architecture.md)
**Nội dung:**
- Tổng quan kiến trúc hệ thống (Clean Architecture)
- Tech stack chi tiết với lý do lựa chọn
- Component design và phân tầng
- Data flow diagrams
- Security, scalability và performance considerations

**Sơ đồ chính:**
- Kiến trúc tổng thể: Frontend (React) → Backend (Go/Echo) → LiveKit (WebRTC) → AI Services (AssemblyAI + Groq)
- Component interaction và communication patterns
- Self-hosted deployment architecture

### 2. [Authentication Flow](./02-authentication-flow.md)
**Nội dung:**
- OAuth2 integration (Google only)
- JWT token management (Access + Refresh tokens)
- Role-based access control (RBAC): Admin, Host, Participant
- Security best practices

**Sequence Diagrams:**
- ✅ OAuth2 Google login flow
- ✅ Token refresh flow
- ✅ Protected API request flow
- ✅ Logout flow
- ✅ Session expiration handling

### 3. [Room Management Flow](./03-room-management-flow.md)
**Nội dung:**
- Tạo và quản lý phòng họp (public/private)
- LiveKit integration và token generation
- Participant management (join, leave, permissions)
- Recording controls (start, stop, pause)
- Screen sharing và media controls
- Real-time WebSocket events

### 4. [AI Analysis Flow](./04-ai-analysis-flow.md)
**Nội dung:**
- Speech-to-Text với AssemblyAI API
- Speaker diarization (phân biệt người nói)
- Content analysis với Groq (LLaMA models)
- Action items extraction và assignment
- Personal participation report generation
- Optional: ClickUp integration

### 5. [Database Schema](./05-database-schema.md)
**Nội dung:**
- PostgreSQL schema design (normalized)
- Table relationships với ERD diagram
- Indexes và query optimization
- Redis cache structure
- Data retention policies
- Monitoring và performance queries

**Core Tables:**
- `users`: User profiles và authentication
- `rooms`: Meeting rooms configuration
- `participants`: Meeting participation records
- `recordings`: Audio/video recordings metadata
- `transcripts`: STT output với timestamps
- `meeting_summaries`: AI-generated summaries
- `action_items`: Extracted tasks
- `participant_reports`: Personal reports
- `notifications`: System notifications

### 6. [API Documentation](./06-api-documentation.md)
**Nội dung:**
- Complete REST API reference
- Request/Response examples với JSON
- Authentication requirements
- Error handling và status codes
- Rate limiting policies
- Webhook endpoints

**API Groups:**
- **Authentication:** `/api/v1/auth/*`
  - POST `/auth/google` - OAuth2 login
  - POST `/auth/refresh` - Refresh tokens
  - POST `/auth/logout` - Invalidate session
- **Users:** `/api/v1/users/*`
- **Rooms:** `/api/v1/rooms/*`
- **Recordings:** `/api/v1/recordings/*`
- **Transcripts:** `/api/v1/meetings/*/transcript`
- **Summaries & Reports:** `/api/v1/meetings/*/summary`
- **Action Items:** `/api/v1/action-items/*`
- **Notifications:** `/api/v1/notifications/*`

### 7. [Deployment Guide](./07-deployment-guide.md)
**Nội dung:**
- Development environment setup
- Docker Compose configuration
- Production deployment checklist
- Environment variables reference
- Monitoring & logging setup (Prometheus + Grafana)
- Backup và disaster recovery strategies
- SSL/TLS configuration với Let's Encrypt

## 🔧 Tech Stack

### Backend
- **Language:** Go 1.21+
- **Framework:** Echo v4 (HTTP router)
- **Architecture:** Clean Architecture (Domain → Use Case → Interface → Infrastructure)
- **Database:** PostgreSQL 15+ (primary data store)
- **Cache:** Redis 7+ (sessions, rate limiting)
- **ORM:** GORM v2 (với raw SQL cho complex queries)

### Frontend
- **Framework:** React 18 + TypeScript
- **State Management:** Zustand (lightweight, simple)
- **UI Library:** Material-UI (MUI v5)
- **WebRTC Client:** LiveKit React SDK
- **Build Tool:** Vite
- **HTTP Client:** Axios

### Real-time Communication
- **Solution:** LiveKit (self-hosted)
- **Protocol:** WebRTC (SFU architecture)
- **Features:** Audio/Video, Screen Share, Recording
- **Scalability:** Horizontal scaling support

### AI Services
- **Speech-to-Text:** AssemblyAI API (với speaker diarization built-in)
- **Content Analysis:** Groq API (LLaMA 3.1/3.2 models)
- **Language Models:** LLaMA 3.1 70B / 3.2 90B (cost-effective, fast inference)
- **Transcription Features:** Word-level timestamps, speaker labels, sentiment analysis

### Infrastructure & DevOps
- **Container:** Docker + Docker Compose
- **Reverse Proxy:** Nginx
- **SSL/TLS:** Let's Encrypt (Certbot)
- **Object Storage:** MinIO (S3-compatible, self-hosted)
- **Monitoring:** Prometheus + Grafana
- **Logging:** Loki + Promtail
- **CI/CD:** GitHub Actions (planned)

## 🚀 Quick Start

### Prerequisites

**Software Requirements:**
```bash
- Go 1.21+ (backend development)
- Node.js 18+ & npm (frontend development)
- Docker & Docker Compose (infrastructure)
- PostgreSQL 15+ (database)
- Redis 7+ (cache)
```

**API Keys Required:**
```bash
- LiveKit API Key & Secret (https://cloud.livekit.io/)
- AssemblyAI API Key (https://www.assemblyai.com/)
- Groq API Key (https://console.groq.com/)
- Google OAuth2 Credentials (https://console.cloud.google.com/)
```

### Development Setup

```bash
# 1. Clone repository
git clone https://github.com/johnquangdev/meeting-assistant.git
cd meeting-assistant

# 2. Start infrastructure services
docker-compose up -d postgres redis minio livekit

# 3. Setup backend
cd backend
cp .env.example .env
# Edit .env with your credentials:
# - DATABASE_URL
# - REDIS_URL
# - LIVEKIT_API_KEY, LIVEKIT_API_SECRET
# - ASSEMBLYAI_API_KEY
# - GROQ_API_KEY
# - GOOGLE_OAUTH_CLIENT_ID, GOOGLE_OAUTH_CLIENT_SECRET
# - JWT_SECRET, JWT_REFRESH_SECRET
# - MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY

go mod download
go run cmd/server/main.go

# 4. Setup frontend (in new terminal)
cd frontend
cp .env.example .env
# Edit .env:
# - VITE_API_URL=http://localhost:8080
# - VITE_LIVEKIT_URL=ws://localhost:7880

npm install
npm run dev

# 5. Access application
# Frontend: http://localhost:5173
# Backend API: http://localhost:8080
# API Docs: http://localhost:8080/swagger
# MinIO Console: http://localhost:9001
# LiveKit: ws://localhost:7880
```

### Docker Compose Services

```yaml
services:
  postgres:
    image: postgres:15-alpine
    ports: 5432:5432
    volumes: ./data/postgres
    
  redis:
    image: redis:7-alpine
    ports: 6379:6379
    
  minio:
    image: minio/minio:latest
    ports: 9000:9000, 9001:9001
    volumes: ./data/minio
    command: server /data --console-address ":9001"
    
  livekit:
    image: livekit/livekit-server:latest
    ports: 7880:7880, 7881:7881
    volumes: ./livekit.yaml:/etc/livekit.yaml
    
  backend:
    build: ./backend
    ports: 8080:8080
    depends_on: [postgres, redis, minio, livekit]
    
  frontend:
    build: ./frontend
    ports: 3000:3000
    depends_on: [backend]
    
  nginx:
    image: nginx:alpine
    ports: 80:80, 443:443
    volumes: ./nginx.conf, ./ssl
    depends_on: [frontend, backend]
```

## 📊 Chức năng chính

### 1. Authentication & User Management ✅
- Google OAuth2 login
- JWT-based authentication (Access + Refresh tokens)
- User profile management
- Role-based access control (Admin, Host, Participant)
- Session management với Redis

### 2. Room Management ✅
- Tạo phòng họp (public/private)
- Scheduled meetings với reminder
- Invite participants qua email
- Host controls: mute all, remove participant, transfer host
- Waiting room (optional)
- Room settings: max participants, recording auto-start

### 3. Real-time Communication ✅
- Audio/Video calls (optimized for 5 participants)
- Screen sharing với audio
- Text chat (in-meeting)
- Connection quality indicators
- Adaptive bitrate
- Network reconnection handling
- Low latency (<200ms target)

### 4. Recording & Transcription ✅
- Cloud recording (audio + video)
- Automatic Speech-to-Text (AssemblyAI API)
- Speaker diarization built-in (phân biệt người nói)
- Word-level timestamps
- Multi-language support (Vietnamese, English)
- Recording playback với transcript sync
- Auto-detect language

### 5. AI Analysis & Insights ✅
- **Meeting Summary:** Groq-generated overview (LLaMA models)
- **Key Points:** Important topics discussed
- **Decisions:** Decisions made during meeting
- **Action Items:** Automatically extracted tasks với assignees
- **Sentiment Analysis:** Overall meeting tone (AssemblyAI built-in)
- **Personal Reports:** Individual participation metrics

### 6. Task Management 📋
- Auto-extracted action items từ transcript
- Task assignment to participants
- Priority levels (High, Medium, Low)
- Due date tracking
- Status updates (Todo, In Progress, Done)
- **Optional:** ClickUp integration

### 7. Notifications 🔔
- Email notifications (meeting invites, reminders)
- In-app notifications
- Meeting start reminders (15 min, 5 min)
- Recording ready alerts
- Task assignment notifications
- Report generation completion

### 8. Reports & Analytics 📈
- **Personal Meeting Reports:**
  - Speaking time percentage
  - Number of contributions
  - Key topics mentioned
  - Assigned action items
- **Export Options:** PDF, DOCX
- **User Statistics:** Meeting history, participation trends

## 🔐 Security Measures

### Authentication & Authorization
- ✅ OAuth2 (Google) - No password storage
- ✅ JWT with short-lived access tokens (15 min)
- ✅ Refresh tokens with rotation
- ✅ HTTP-only cookies for refresh tokens
- ✅ RBAC with permission checks

### Network Security
- ✅ HTTPS/TLS encryption (Let's Encrypt)
- ✅ CORS configuration (whitelist origins)
- ✅ Rate limiting (per IP, per user)
- ✅ DDoS protection (Nginx)
- ✅ WebSocket secure connections (WSS)

### Data Security
- ✅ Input validation (all endpoints)
- ✅ SQL injection prevention (parameterized queries)
- ✅ XSS protection (sanitized output)
- ✅ CSRF protection (tokens)
- ✅ Encrypted sensitive data at rest
- ✅ Secure file upload validation

### API Security
- ✅ API key rotation
- ✅ Request signing
- ✅ IP whitelisting (admin endpoints)
- ✅ Audit logging

## 📚 Tài liệu tham khảo

### Official Documentation
- [LiveKit Docs](https://docs.livekit.io/) - WebRTC platform
- [AssemblyAI API](https://www.assemblyai.com/docs) - Speech-to-Text & Diarization
- [Groq API](https://console.groq.com/docs) - Fast LLM inference
- [Echo Framework](https://echo.labstack.com/guide/) - Go web framework
- [PostgreSQL](https://www.postgresql.org/docs/) - Database
- [Redis](https://redis.io/documentation) - Cache & sessions
- [React](https://react.dev/) - Frontend framework
- [Material-UI](https://mui.com/) - UI components

### Learning Resources
- [Clean Architecture in Go](https://github.com/bxcodec/go-clean-arch)
- [WebRTC for Beginners](https://webrtc.org/getting-started/overview)
- [JWT Best Practices](https://datatracker.ietf.org/doc/html/rfc8725)
- [System Design Primer](https://github.com/donnemartin/system-design-primer)

### Tools & Libraries
- [GORM](https://gorm.io/) - Go ORM
- [Zustand](https://github.com/pmndrs/zustand) - State management
- [Axios](https://axios-http.com/) - HTTP client
- [Vite](https://vitejs.dev/) - Build tool


## 📞 Liên hệ

**Nhóm sinh viên:**
- **Nguyễn Minh Quang** (24410217)
  - Email: 24410217@student.uit.edu.vn
  - GitHub: [@johnquangdev](https://github.com/johnquangdev)
  
- **Trần Đức Minh** (24410197)
  - Email: 24410197@student.uit.edu.vn

**Giảng viên hướng dẫn:**
- **ThS. Đặng Văn Thìn**
  - Email: thindv@uit.edu.vn
  - Khoa Công Nghệ Phần Mềm - UIT

**Thông tin dự án:**
- **Môn học:** Đồ án chuyên ngành
- **Học kỳ:** 1 - Năm học 2024-2025
- **Trường:** Đại học Công Nghệ Thông Tin - ĐHQG TP.HCM

**Copyright © 2024 - Nguyễn Minh Quang, Trần Đức Minh**

---

**Last Updated:** November 2024

**Version:** 1.0.0-alpha (Documentation Phase)

**Repository:** [github.com/johnquangdev/meeting-assistant](https://github.com/johnquangdev/meeting-assistant)
