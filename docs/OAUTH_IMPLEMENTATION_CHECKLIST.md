# Google OAuth2 Implementation Checklist

## ✅ Completed Components

### 1. Domain Layer
- ✅ `internal/domain/entities/user.go` - User entity với OAuth fields
- ✅ `internal/domain/entities/session.go` - Session entity
- ✅ `internal/domain/entities/errors.go` - Domain errors
- ✅ `internal/domain/repositories/user_repository.go` - User repository interface
- ✅ `internal/domain/repositories/session_repository.go` - Session repository interface

### 2. Infrastructure Layer
- ✅ `internal/infrastructure/external/oauth/google.go` - Google OAuth provider
- ✅ `internal/infrastructure/external/oauth/state.go` - State manager for CSRF protection
- ✅ `internal/infrastructure/http/middleware/auth_middleware.go` - Authentication middleware
- ✅ `internal/infrastructure/http/routes/auth_routes.go` - Auth routes setup

### 3. Use Case Layer
- ✅ `internal/usecase/auth/oauth_service.go` - OAuth business logic

### 4. Adapter Layer
- ✅ `internal/adapter/handler/auth_handler.go` - HTTP handlers

### 5. Configuration
- ✅ `pkg/config/config.go` - Configuration management
- ✅ `.env.example` - Environment variables template

### 6. Documentation
- ✅ `docs/GOOGLE_OAUTH_SETUP.md` - Complete setup guide

## 🔨 TODO: Implementation Steps

### Step 1: Implement Repository Layer (PostgreSQL)

Cần tạo PostgreSQL implementations cho repositories:

**File cần tạo:**
```
internal/adapter/repository/
  ├── postgres_user_repository.go
  └── postgres_session_repository.go
```

**Code structure:**
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

// Implement all methods from UserRepository interface
func (r *PostgresUserRepository) Create(ctx context.Context, user *entities.User) error {
    query := `
        INSERT INTO users (
            id, email, name, role, is_active,
            oauth_provider, oauth_id, oauth_refresh_token,
            avatar_url, bio, timezone, language,
            is_email_verified, notification_preferences, meeting_preferences,
            created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
    `
    // ... implement
}

// ... implement other methods
```

### Step 2: Database Connection

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
    dsn := cfg.GetDatabaseDSN()
    
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }
    
    db.SetMaxOpenConns(cfg.Database.MaxConns)
    db.SetMaxIdleConns(cfg.Database.MinConns)
    
    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }
    
    return db, nil
}
```

### Step 3: Dependency Injection

**File: `cmd/api/main.go`**

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "time"

    "github.com/johnquangdev/meeting-assistant/internal/adapter/handler"
    "github.com/johnquangdev/meeting-assistant/internal/adapter/repository"
    "github.com/johnquangdev/meeting-assistant/internal/infrastructure/database"
    "github.com/johnquangdev/meeting-assistant/internal/infrastructure/external/oauth"
    "github.com/johnquangdev/meeting-assistant/internal/infrastructure/http/middleware"
    "github.com/johnquangdev/meeting-assistant/internal/infrastructure/http/routes"
    "github.com/johnquangdev/meeting-assistant/internal/usecase/auth"
    "github.com/johnquangdev/meeting-assistant/pkg/config"
)

func main() {
    // Load config
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Initialize database
    db, err := database.NewPostgresConnection(cfg)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer db.Close()

    // Initialize repositories
    userRepo := repository.NewPostgresUserRepository(db)
    sessionRepo := repository.NewPostgresSessionRepository(db)

    // Initialize OAuth providers
    googleProvider := oauth.NewGoogleProvider(
        cfg.OAuth.Google.ClientID,
        cfg.OAuth.Google.ClientSecret,
        cfg.OAuth.Google.RedirectURL,
    )
    stateManager := oauth.NewStateManager()

    // Initialize use cases
    oauthService := auth.NewOAuthService(
        userRepo,
        sessionRepo,
        googleProvider,
        stateManager,
    )

    // Initialize handlers
    authHandler := handler.NewAuthHandler(oauthService)

    // Initialize middleware
    authMiddleware := middleware.NewAuthMiddleware(oauthService)

    // Setup routes
    mux := http.NewServeMux()
    routes.SetupAuthRoutes(mux, authHandler, authMiddleware)

    // Add CORS middleware
    handler := enableCORS(mux, cfg.Server.AllowedOrigins)

    // Create server
    srv := &http.Server{
        Addr:         cfg.Server.Host + ":" + cfg.Server.Port,
        Handler:      handler,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    // Start server
    go func() {
        log.Printf("Server starting on %s", srv.Addr)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server failed: %v", err)
        }
    }()

    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt)
    <-quit

    log.Println("Shutting down server...")
    ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Server.ShutdownTimeout)*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("Server forced to shutdown: %v", err)
    }

    log.Println("Server exited")
}

func enableCORS(next http.Handler, allowedOrigins []string) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        origin := r.Header.Get("Origin")
        for _, allowed := range allowedOrigins {
            if origin == allowed {
                w.Header().Set("Access-Control-Allow-Origin", origin)
                break
            }
        }
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        w.Header().Set("Access-Control-Allow-Credentials", "true")

        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

### Step 4: Install Dependencies

```bash
# PostgreSQL driver
go get github.com/lib/pq

# JSON encoding
go get github.com/json-iterator/go  # (optional, faster than standard)
```

### Step 5: Update Makefile

Add commands to Makefile:

```makefile
# Run application
.PHONY: run
run:
	go run cmd/api/main.go

# Build application
.PHONY: build
build:
	go build -o bin/api cmd/api/main.go

# Run with hot reload (requires air)
.PHONY: dev
dev:
	air

# Install air for hot reload
.PHONY: install-air
install-air:
	go install github.com/cosmtrek/air@latest
```

### Step 6: Create .air.toml (Optional - Hot Reload)

```toml
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = []
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./cmd/api"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html"]
  include_file = []
  kill_delay = "0s"
  log = "build-errors.log"
  poll = false
  poll_interval = 0
  rerun = false
  rerun_delay = 500
  send_interrupt = false
  stop_on_error = false

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  main_only = false
  time = false

[misc]
  clean_on_exit = false

[screen]
  clear_on_rebuild = false
  keep_scroll = true
```

## 📋 Testing Checklist

### Manual Testing

- [ ] Start Docker services
- [ ] Run migrations
- [ ] Configure `.env` with Google OAuth credentials
- [ ] Start application
- [ ] Test `/api/v1/auth/google/login` endpoint
- [ ] Complete OAuth flow in browser
- [ ] Verify user created in database
- [ ] Test `/api/v1/auth/me` with access token
- [ ] Test `/api/v1/auth/logout`
- [ ] Test token refresh

### Unit Testing

Files to create:
```
internal/usecase/auth/oauth_service_test.go
internal/adapter/handler/auth_handler_test.go
internal/infrastructure/external/oauth/google_test.go
```

### Integration Testing

Files to create:
```
tests/integration/auth_test.go
tests/integration/oauth_flow_test.go
```

## 🚀 Next Features to Implement

1. **Email/Password Authentication** (optional)
2. **Multiple OAuth Providers** (GitHub, Microsoft)
3. **Two-Factor Authentication**
4. **Role-Based Access Control (RBAC)**
5. **API Rate Limiting**
6. **Audit Logging**
7. **Session Management UI**
8. **Password Reset Flow** (if email auth)

## 📖 API Endpoints Summary

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| GET | `/api/v1/auth/google/login` | Get Google OAuth URL | No |
| GET | `/api/v1/auth/google/callback` | OAuth callback | No |
| POST | `/api/v1/auth/refresh` | Refresh access token | No |
| GET | `/api/v1/auth/me` | Get current user | Yes |
| POST | `/api/v1/auth/logout` | Logout user | Yes |

## 🔐 Security Considerations

- [x] State parameter for CSRF protection
- [x] Token hashing (SHA256)
- [x] Session expiration
- [x] Secure token storage
- [ ] Rate limiting on auth endpoints
- [ ] Brute force protection
- [ ] IP whitelisting (optional)
- [ ] MFA/2FA support
- [ ] Session device tracking
- [ ] Suspicious login detection

## 📚 Documentation Links

- [Google OAuth Setup Guide](./GOOGLE_OAUTH_SETUP.md)
- [API Documentation](./06-api-documentation.md)
- [Database Schema](./05-database-schema.md)
- [System Architecture](./01-system-architecture.md)
