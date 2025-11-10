# Google OAuth2 Implementation - Quick Start

## 📦 Những gì đã tạo

### 1. **Domain Layer** (Business Logic Core)
```
internal/domain/
├── entities/
│   ├── user.go          # User entity với OAuth fields
│   ├── session.go       # Session management
│   └── errors.go        # Domain errors
└── repositories/
    ├── user_repository.go     # User data interface
    └── session_repository.go  # Session data interface
```

### 2. **Infrastructure Layer** (External Services)
```
internal/infrastructure/
├── external/oauth/
│   ├── google.go        # Google OAuth provider
│   └── state.go         # CSRF protection
└── http/
    ├── middleware/
    │   └── auth_middleware.go  # Authentication middleware
    └── routes/
        └── auth_routes.go      # Route definitions
```

### 3. **Use Case Layer** (Application Logic)
```
internal/usecase/auth/
└── oauth_service.go     # OAuth business logic
```

### 4. **Adapter Layer** (HTTP Handlers)
```
internal/adapter/handler/
└── auth_handler.go      # HTTP request handlers
```

### 5. **Configuration**
```
pkg/config/
└── config.go            # App configuration

.env.example             # Environment template
```

### 6. **Documentation**
```
docs/
├── GOOGLE_OAUTH_SETUP.md           # Chi tiết setup Google OAuth
└── OAUTH_IMPLEMENTATION_CHECKLIST.md  # Implementation checklist
```

## 🎯 Bước tiếp theo

### Bước 1: Setup Google Cloud (5-10 phút)

1. **Tạo Google Cloud Project**
   - Vào https://console.cloud.google.com/
   - Tạo project mới

2. **Enable APIs**
   - Enable Google+ API hoặc Google Identity API

3. **Tạo OAuth Credentials**
   - Tạo OAuth 2.0 Client ID
   - Redirect URI: `http://localhost:8080/api/v1/auth/google/callback`
   - Copy Client ID và Client Secret

4. **Configure .env**
   ```bash
   cp .env.example .env
   # Điền GOOGLE_CLIENT_ID và GOOGLE_CLIENT_SECRET
   ```

👉 **Chi tiết:** Xem file `docs/GOOGLE_OAUTH_SETUP.md`

### Bước 2: Implement Repository Layer (30-45 phút)

Tạo PostgreSQL implementations:

**File cần tạo:**
```
internal/adapter/repository/
├── postgres_user_repository.go
└── postgres_session_repository.go
```

**Template:**
```go
// postgres_user_repository.go
package repository

import (
    "context"
    "database/sql"
    "github.com/google/uuid"
    "github.com/johnquangdev/meeting-assistant/internal/domain/entities"
    "github.com/johnquangdev/meeting-assistant/internal/domain/repositories"
)

type PostgresUserRepository struct {
    db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) repositories.UserRepository {
    return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *entities.User) error {
    query := `
        INSERT INTO users (
            id, email, name, role, is_active,
            oauth_provider, oauth_id, oauth_refresh_token,
            avatar_url, timezone, language,
            is_email_verified, notification_preferences, meeting_preferences,
            created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
    `
    _, err := r.db.ExecContext(ctx, query,
        user.ID, user.Email, user.Name, user.Role, user.IsActive,
        user.OAuthProvider, user.OAuthID, user.OAuthRefreshToken,
        user.AvatarURL, user.Timezone, user.Language,
        user.IsEmailVerified, user.NotificationPreferences, user.MeetingPreferences,
        user.CreatedAt, user.UpdatedAt,
    )
    return err
}

// Implement các methods khác tương tự
```

### Bước 3: Database Connection (10 phút)

**File: `internal/infrastructure/database/postgres.go`**

```go
package database

import (
    "database/sql"
    "fmt"
    _ "github.com/lib/pq"
    "github.com/johnquangdev/meeting-assistant/pkg/config"
)

func NewPostgresConnection(cfg *config.Config) (*sql.DB, error) {
    db, err := sql.Open("postgres", cfg.GetDatabaseDSN())
    if err != nil {
        return nil, err
    }
    
    db.SetMaxOpenConns(cfg.Database.MaxConns)
    db.SetMaxIdleConns(cfg.Database.MinConns)
    
    if err := db.Ping(); err != nil {
        return nil, err
    }
    
    return db, nil
}
```

**Install dependencies:**
```bash
go get github.com/lib/pq
```

### Bước 4: Wire Everything in main.go (15 phút)

Xem full example trong `docs/OAUTH_IMPLEMENTATION_CHECKLIST.md` section "Step 3: Dependency Injection"

**Key points:**
1. Load config
2. Connect database
3. Initialize repositories
4. Setup OAuth providers
5. Create services
6. Setup handlers & routes
7. Start HTTP server

### Bước 5: Test (10 phút)

```bash
# Start services
docker-compose up -d

# Run migrations
make migrate-up

# Start app
go run cmd/api/main.go

# Test OAuth flow
curl http://localhost:8080/api/v1/auth/google/login
```

## 🏗️ Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                     HTTP Layer                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │   Routes     │→ │   Handler    │→ │  Middleware  │ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
└────────────────────────┬────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────┐
│                   Use Case Layer                        │
│  ┌──────────────────────────────────────────────────┐  │
│  │            OAuth Service                         │  │
│  │  - GetGoogleAuthURL()                            │  │
│  │  - HandleGoogleCallback()                        │  │
│  │  - ValidateSession()                             │  │
│  └──────────────────────────────────────────────────┘  │
└────────────────────────┬────────────────────────────────┘
                         │
         ┌───────────────┴───────────────┐
         │                               │
┌────────▼────────┐            ┌─────────▼──────────┐
│  Infrastructure │            │   Domain Layer     │
│                 │            │                    │
│ ┌─────────────┐ │            │ ┌────────────────┐ │
│ │   Google    │ │            │ │   Entities     │ │
│ │   OAuth     │ │            │ │  - User        │ │
│ │  Provider   │ │            │ │  - Session     │ │
│ └─────────────┘ │            │ └────────────────┘ │
│                 │            │                    │
│ ┌─────────────┐ │            │ ┌────────────────┐ │
│ │  Postgres   │ │◄───────────┤ │  Repositories  │ │
│ │ Repository  │ │            │ │  (Interfaces)  │ │
│ └─────────────┘ │            │ └────────────────┘ │
└─────────────────┘            └────────────────────┘
```

## 📊 OAuth Flow

```
1. Client → GET /api/v1/auth/google/login
   ↓
2. Server generates state & returns Google Auth URL
   ↓
3. Client redirects to Google
   ↓
4. User authenticates with Google
   ↓
5. Google → Redirect to /api/v1/auth/google/callback?code=xxx&state=xxx
   ↓
6. Server:
   - Validates state
   - Exchanges code for tokens
   - Gets user info from Google
   - Creates/updates user in DB
   - Creates session
   - Returns access token
   ↓
7. Client stores token & uses for API calls
   ↓
8. Protected endpoints validate token via middleware
```

## 🔑 API Endpoints

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| GET | `/api/v1/auth/google/login` | Get Google OAuth URL | No |
| GET | `/api/v1/auth/google/callback` | OAuth callback | No |
| POST | `/api/v1/auth/refresh` | Refresh token | No |
| GET | `/api/v1/auth/me` | Get current user | ✅ Yes |
| POST | `/api/v1/auth/logout` | Logout | ✅ Yes |

## 🎓 Learning Resources

1. **Google OAuth Setup:** `docs/GOOGLE_OAUTH_SETUP.md`
2. **Implementation Checklist:** `docs/OAUTH_IMPLEMENTATION_CHECKLIST.md`
3. **System Architecture:** `docs/01-system-architecture.md`
4. **Database Schema:** `docs/05-database-schema.md`

## ❓ FAQ

**Q: Tại sao không dùng JWT?**
A: Session-based auth đơn giản hơn và dễ revoke. JWT requires more complex logic for revocation.

**Q: State manager có thread-safe không?**
A: In-memory version không suitable cho production. Nên dùng Redis trong production.

**Q: Làm sao test mà không có Google credentials?**
A: Tạo mock implementations của GoogleProvider interface cho unit tests.

**Q: Có cần HTTPS cho development không?**
A: Không bắt buộc, nhưng production PHẢI dùng HTTPS.

## 🚀 Ready to Code!

Bắt đầu với bước 2 (Implement Repository Layer) và follow checklist trong `docs/OAUTH_IMPLEMENTATION_CHECKLIST.md`

Good luck! 🎉
