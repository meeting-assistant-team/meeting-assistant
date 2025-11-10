# System Architecture

## Tổng quan hệ thống

**Meeting Assistant** là một ứng dụng web meeting với tích hợp AI để ghi âm, phân tích và tóm tắt cuộc họp tự động.

## Tech Stack

### Backend
- **Language**: Golang 1.21+
- **Framework**: Echo v4
- **Architecture**: Clean Architecture (Layered)
- **Database**: PostgreSQL 15+
- **Cache**: Redis 7+
- **ORM**: GORM / sqlx

### Frontend
- **Framework**: React + TypeScript
- **State Management**: Redux Toolkit / Zustand
- **UI Library**: Material-UI / Ant Design
- **WebRTC Client**: LiveKit Client SDK

### Real-time Communication
- **Solution**: LiveKit
- **Protocol**: WebRTC
- **Signaling**: LiveKit Server

### AI Services
- **Speech-to-Text**: OpenAI Whisper API / Local Whisper
- **Text Analysis**: OpenAI GPT-4 API
- **Summary & Action Items**: Custom NLP + GPT

### Infrastructure
- **Containerization**: Docker
- **Orchestration**: Docker Compose (MVP) / Kubernetes (Production)
- **Reverse Proxy**: Nginx
- **SSL/TLS**: Let's Encrypt
- **Monitoring**: Prometheus + Grafana

## Kiến trúc tổng thể

```
┌─────────────────────────────────────────────────────────────┐
│                         Client Layer                         │
│  ┌────────────────────────────────────────────────────────┐ │
│  │         React Frontend (TypeScript)                    │ │
│  │  - Meeting UI                                          │ │
│  │  - Dashboard                                           │ │
│  │  - User Management                                     │ │
│  │  - LiveKit Client SDK                                  │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ HTTPS/WSS
                              │
┌─────────────────────────────▼─────────────────────────────────┐
│                      API Gateway / Nginx                       │
└─────────────────────────────┬─────────────────────────────────┘
                              │
                ┌─────────────┴──────────────┐
                │                            │
┌───────────────▼────────────┐  ┌───────────▼──────────────┐
│   Backend Services (Go)    │  │   LiveKit Server         │
│                            │  │                          │
│  ┌──────────────────────┐  │  │  - SFU Media Router     │
│  │  Auth Service        │  │  │  - WebRTC Signaling     │
│  │  - OAuth2           │  │  │  - Room Management      │
│  │  - JWT              │  │  │  - Recording            │
│  └──────────────────────┘  │  └──────────────────────────┘
│                            │              │
│  ┌──────────────────────┐  │              │
│  │  Room Service        │◄─┼──────────────┘
│  │  - Create/Join      │  │   WebHooks
│  │  - Manage           │  │
│  └──────────────────────┘  │
│                            │
│  ┌──────────────────────┐  │
│  │  Recording Service   │  │
│  │  - Audio Capture    │  │
│  │  - Storage          │  │
│  └──────────────────────┘  │
│                            │
│  ┌──────────────────────┐  │
│  │  AI Service          │  │
│  │  - STT              │  │
│  │  - Analysis         │  │
│  │  - Summary          │  │
│  └──────────────────────┘  │
│                            │
│  ┌──────────────────────┐  │
│  │  Report Service      │  │
│  │  - Dashboard        │  │
│  │  - Notifications    │  │
│  └──────────────────────┘  │
│                            │
│  ┌──────────────────────┐  │
│  │  Integration Service │  │
│  │  - ClickUp API      │  │
│  └──────────────────────┘  │
└────────────┬───────────────┘
             │
┌────────────▼───────────────┐
│    Data Layer              │
│                            │
│  ┌──────────────────────┐  │
│  │  PostgreSQL          │  │
│  │  - Users             │  │
│  │  - Rooms             │  │
│  │  - Meetings          │  │
│  │  - Transcripts       │  │
│  │  - Reports           │  │
│  └──────────────────────┘  │
│                            │
│  ┌──────────────────────┐  │
│  │  Redis               │  │
│  │  - Sessions          │  │
│  │  - Cache             │  │
│  └──────────────────────┘  │
│                            │
│  ┌──────────────────────┐  │
│  │  S3/MinIO            │  │
│  │  - Audio Files       │  │
│  │  - Recordings        │  │
│  └──────────────────────┘  │
└────────────────────────────┘

┌────────────────────────────┐
│   External Services        │
│                            │
│  - OpenAI API (Whisper)    │
│  - OpenAI API (GPT-4)      │
│  - ClickUp API             │
│  - Email/SMS Service       │
└────────────────────────────┘
```

## Component Details

### 1. Frontend (React)

**Responsibilities:**
- User interface rendering
- WebRTC client management (LiveKit SDK)
- State management
- Real-time updates via WebSocket
- Local media handling (camera/microphone)

**Key Pages:**
- `/login` - OAuth2 authentication
- `/dashboard` - User dashboard
- `/room/:id` - Meeting room
- `/reports` - Meeting reports
- `/profile` - User profile

### 2. Backend Services (Golang + Echo)

**Architecture Pattern:** Clean Architecture

```
backend/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── domain/                  # Enterprise Business Rules
│   │   ├── entities/            # Core entities (User, Room, Meeting)
│   │   ├── repositories/        # Repository interfaces
│   │   └── errors/              # Domain errors
│   │
│   ├── usecase/                 # Application Business Rules
│   │   ├── auth/                # Auth use cases
│   │   ├── room/                # Room use cases
│   │   ├── recording/           # Recording use cases
│   │   ├── ai/                  # AI processing use cases
│   │   └── report/              # Report use cases
│   │
│   ├── adapter/                 # Interface Adapters
│   │   ├── handler/             # HTTP handlers (Echo)
│   │   │   ├── auth_handler.go
│   │   │   ├── room_handler.go
│   │   │   └── middleware/
│   │   ├── repository/          # Repository implementations
│   │   │   ├── postgres/        # PostgreSQL repos
│   │   │   └── redis/           # Redis repos
│   │   └── presenter/           # Response formatters
│   │
│   ├── infrastructure/          # Frameworks & Drivers
│   │   ├── http/                # Echo server setup
│   │   ├── database/            # DB connections
│   │   ├── cache/               # Redis client
│   │   ├── storage/             # MinIO client
│   │   └── external/            # External APIs
│   │       ├── livekit/         # LiveKit client
│   │       ├── openai/          # OpenAI client
│   │       └── oauth/           # Google OAuth
│   │
│   ├── pkg/                     # Shared packages (reusable)
│   │   ├── jwt/                 # JWT token utilities
│   │   ├── logger/              # Structured logging
│   │   ├── validator/           # Input validation
│   │   └── config/              # Configuration loader
│   │
│   └── utils/                   # Helper functions
│       ├── string.go            # String helpers
│       ├── time.go              # Time/date formatters
│       └── crypto.go            # Encryption helpers
│
├── migrations/                  # Database migrations
├── docs/                        # API documentation (Swagger)
├── go.mod
└── go.sum
```

**Clean Architecture Layers:**

1. **Domain Layer (Innermost)**
   - Pure business logic
   - No external dependencies
   - Entities and repository interfaces
   ```go
   // domain/entities/room.go
   type Room struct {
       ID        uuid.UUID
       Name      string
       HostID    uuid.UUID
       Status    RoomStatus
       CreatedAt time.Time
   }
   
   // domain/repositories/room_repository.go
   type RoomRepository interface {
       Create(ctx context.Context, room *Room) error
       FindByID(ctx context.Context, id uuid.UUID) (*Room, error)
       Update(ctx context.Context, room *Room) error
   }
   ```

2. **Use Case Layer**
   - Application-specific business rules
   - Orchestrates data flow
   - Depends only on domain layer
   ```go
   // usecase/room/create_room.go
   type CreateRoomUseCase struct {
       roomRepo      domain.RoomRepository
       livekitClient livekit.Client
   }
   
   func (uc *CreateRoomUseCase) Execute(ctx context.Context, req CreateRoomRequest) (*Room, error) {
       // Business logic here
   }
   ```

3. **Adapter Layer**
   - Converts data between use cases and external formats
   - HTTP handlers (Echo)
   - Repository implementations
   ```go
   // adapter/handler/room_handler.go
   type RoomHandler struct {
       createRoomUC usecase.CreateRoomUseCase
   }
   
   func (h *RoomHandler) CreateRoom(c echo.Context) error {
       var req CreateRoomRequest
       if err := c.Bind(&req); err != nil {
           return c.JSON(400, ErrorResponse{...})
       }
       room, err := h.createRoomUC.Execute(c.Request().Context(), req)
       // ...
   }
   ```

4. **Infrastructure Layer (Outermost)**
   - Framework implementations
   - Database drivers
   - External service clients
   ```go
   // infrastructure/http/server.go
   func NewEchoServer(cfg *config.Config) *echo.Echo {
       e := echo.New()
       e.Use(middleware.Logger())
       e.Use(middleware.Recover())
       // Setup routes
       return e
   }
   ```

**Services Breakdown:**

#### Auth Service (Use Cases)
- OAuth2 integration (Google)
- JWT token generation/validation
- User session management (Redis)
- Role-based access control (Host/Participant)

**Key Files:**
- `usecase/auth/login.go`
- `usecase/auth/refresh_token.go`
- `adapter/handler/auth_handler.go`
- `infrastructure/external/oauth/google.go`

#### Room Service (Use Cases)
- Room CRUD operations
- Participant management
- Room state synchronization
- Integration with LiveKit

**Key Files:**
- `usecase/room/create_room.go`
- `usecase/room/join_room.go`
- `usecase/room/manage_participants.go`
- `infrastructure/external/livekit/client.go`

#### Recording Service (Use Cases)
- Audio stream capture from LiveKit
- File storage management (MinIO)
- Recording lifecycle
- Cleanup jobs

**Key Files:**
- `usecase/recording/start_recording.go`
- `usecase/recording/stop_recording.go`
- `infrastructure/storage/minio.go`

#### AI Service (Use Cases)
- Audio preprocessing
- Speech-to-Text via Whisper
- Transcript processing
- GPT-4 analysis for:
  - Meeting summary
  - Action items extraction
  - Speaker identification
  - Key points

**Key Files:**
- `usecase/ai/transcribe.go`
- `usecase/ai/analyze_meeting.go`
- `infrastructure/external/openai/whisper.go`
- `infrastructure/external/openai/gpt4.go`

#### Report Service (Use Cases)
- Report generation
- Dashboard data aggregation
- User-specific reports
- Export functionality

**Key Files:**
- `usecase/report/generate_report.go`
- `usecase/report/export_pdf.go`
- `adapter/handler/report_handler.go`

### 3. LiveKit Server

**Responsibilities:**
- WebRTC SFU (Selective Forwarding Unit)
- Real-time audio/video routing
- Participant management
- Recording capabilities
- Room webhooks

**Key Features:**
- Low latency (<200ms)
- Scalable to 5-10 participants
- Built-in recording
- WebHook notifications

### 4. Database Layer

#### PostgreSQL Schema

**Tables:**
- `users` - User accounts
- `rooms` - Meeting rooms
- `participants` - Room participants
- `meetings` - Meeting sessions
- `recordings` - Audio recordings
- `transcripts` - STT results
- `action_items` - Extracted tasks
- `reports` - Generated reports
- `notifications` - User notifications

#### Redis Cache

**Keys:**
- `session:{token}` - User sessions
- `room:{id}:participants` - Active participants
- `room:{id}:state` - Room state
- `user:{id}:active_meeting` - Current meeting

### 5. Storage (S3/MinIO)

**Buckets:**
- `recordings/` - Raw audio files
- `processed/` - Processed audio
- `exports/` - Report exports
- `avatars/` - User avatars

## Data Flow

### Meeting Flow

```
User → Frontend → Backend API → LiveKit → Media Streaming
                      ↓
                 PostgreSQL (metadata)
                      ↓
                 Recording Service → S3
                      ↓
                 AI Service (STT) → Transcript
                      ↓
                 AI Service (GPT) → Summary + Action Items
                      ↓
                 Report Service → Dashboard
```

## Security Considerations

### Authentication & Authorization
- OAuth2 for user login
- JWT for API authentication
- Refresh token rotation
- Role-based permissions

### Data Security
- TLS/SSL for all connections
- Encrypted storage for sensitive data
- Secure room access tokens
- CORS configuration

### Privacy
- Audio data encryption
- GDPR compliance considerations
- User data deletion
- Recording consent

## Scalability Strategy

### Phase 1 (MVP - Current)
- Single server deployment
- 5-10 concurrent users
- Docker Compose
- Vertical scaling

### Phase 2 (Future)
- Horizontal scaling
- Load balancing
- Kubernetes deployment
- 50-100 concurrent users

### Phase 3 (Production)
- Multi-region deployment
- CDN integration
- Advanced monitoring
- Auto-scaling

## Monitoring & Logging

### Metrics
- API response times
- WebRTC connection quality
- Database query performance
- AI service latency
- Error rates

### Logging
- Structured logging (JSON)
- Centralized log aggregation
- Log levels (DEBUG, INFO, WARN, ERROR)
- Request tracing

### Alerting
- Prometheus alerts
- Slack/Email notifications
- Critical error monitoring
- Resource usage thresholds

## Deployment Architecture

```
┌─────────────────────────────────────────┐
│           Production Server             │
│                                         │
│  ┌───────────────────────────────────┐  │
│  │  Nginx (Reverse Proxy)            │  │
│  │  - SSL Termination                │  │
│  │  - Load Balancing                 │  │
│  └───────────────────────────────────┘  │
│                  │                      │
│         ┌────────┴────────┐             │
│         │                 │             │
│  ┌──────▼─────┐   ┌──────▼─────┐       │
│  │  Backend   │   │  Frontend  │       │
│  │  (Docker)  │   │  (Docker)  │       │
│  └────────────┘   └────────────┘       │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │  LiveKit Server (Docker)        │   │
│  └─────────────────────────────────┘   │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │  PostgreSQL (Docker)            │   │
│  └─────────────────────────────────┘   │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │  Redis (Docker)                 │   │
│  └─────────────────────────────────┘   │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │  MinIO (Docker)                 │   │
│  └─────────────────────────────────┘   │
└─────────────────────────────────────────┘
```

## Technology Decisions

### Why LiveKit?
- ✅ Production-ready WebRTC infrastructure
- ✅ Built-in recording capabilities
- ✅ Scalable SFU architecture
- ✅ Easy integration with Golang
- ✅ Active community and documentation

### Why Echo Framework?
- ✅ High performance (faster than Gin)
- ✅ Minimalist and flexible
- ✅ Built-in middleware support
- ✅ Context-based request handling
- ✅ Easy to test and maintain
- ✅ Excellent documentation

### Why Clean Architecture?
- ✅ Independent of frameworks (can switch Echo easily)
- ✅ Testable business logic
- ✅ Independent of UI/Database
- ✅ Clear separation of concerns
- ✅ Maintainable and scalable codebase
- ✅ Easier onboarding for new developers

### Why Golang?
- ✅ High performance
- ✅ Excellent concurrency support
- ✅ Native WebRTC libraries
- ✅ Small binary size
- ✅ Fast compilation

### Why PostgreSQL?
- ✅ ACID compliance
- ✅ JSON support
- ✅ Full-text search
- ✅ Strong community
- ✅ Reliable and mature

### Why React?
- ✅ Component-based architecture
- ✅ Large ecosystem
- ✅ TypeScript support
- ✅ LiveKit SDK availability
- ✅ Team familiarity

## Development Environment

### Local Setup
```bash
# Clone repository
git clone <repo-url>
cd meeting-assistant

# Start infrastructure services with Docker Compose
docker-compose -f docker-compose.dev.yml up -d

# Run backend (with hot reload)
cd backend
go mod download
go install github.com/cosmtrek/air@latest
air  # Hot reload server

# Run frontend
cd frontend
npm install
npm start
```

### Backend Project Structure Example

```go
// cmd/server/main.go
package main

import (
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
    "meeting-assistant/internal/infrastructure/http"
    "meeting-assistant/internal/pkg/config"
)

func main() {
    cfg := config.Load()
    
    e := echo.New()
    e.Use(middleware.Logger())
    e.Use(middleware.Recover())
    e.Use(middleware.CORS())
    
    http.SetupRoutes(e, cfg)
    
    e.Logger.Fatal(e.Start(":8080"))
}
```

```go
// internal/infrastructure/http/routes.go
package http

import (
    "github.com/labstack/echo/v4"
    "meeting-assistant/internal/adapter/handler"
)

func SetupRoutes(e *echo.Echo, cfg *config.Config) {
    api := e.Group("/api/v1")
    
    // Auth routes
    authHandler := handler.NewAuthHandler(cfg)
    api.POST("/auth/google", authHandler.GoogleLogin)
    api.POST("/auth/refresh", authHandler.RefreshToken)
    
    // Protected routes
    protected := api.Group("", middleware.JWTMiddleware())
    
    // Room routes
    roomHandler := handler.NewRoomHandler(cfg)
    protected.POST("/rooms", roomHandler.CreateRoom)
    protected.GET("/rooms/:id", roomHandler.GetRoom)
    protected.POST("/rooms/:id/join", roomHandler.JoinRoom)
}
```

### Environment Variables
```env
# Backend
DATABASE_URL=postgresql://...
REDIS_URL=redis://...
LIVEKIT_API_KEY=...
LIVEKIT_API_SECRET=...
OPENAI_API_KEY=...
JWT_SECRET=...

# Frontend
REACT_APP_API_URL=http://localhost:8080
REACT_APP_LIVEKIT_URL=ws://localhost:7880
```

## API Documentation

API documentation được tạo với Swagger/OpenAPI và có thể truy cập tại:
- Development: `http://localhost:8080/swagger`
- Production: `https://api.meetingassistant.com/swagger`

## References

- [LiveKit Documentation](https://docs.livekit.io/)
- [OpenAI Whisper](https://platform.openai.com/docs/guides/speech-to-text)
- [Go Web Development](https://go.dev/doc/)
- [React Documentation](https://react.dev/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
