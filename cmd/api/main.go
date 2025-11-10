package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/johnquangdev/meeting-assistant/internal/adapter/handler"
	"github.com/johnquangdev/meeting-assistant/internal/adapter/repository"
	"github.com/johnquangdev/meeting-assistant/internal/infrastructure/cache"
	"github.com/johnquangdev/meeting-assistant/internal/infrastructure/database"
	"github.com/johnquangdev/meeting-assistant/internal/infrastructure/external/oauth"
	"github.com/johnquangdev/meeting-assistant/internal/usecase/auth"
	"github.com/johnquangdev/meeting-assistant/pkg/config"
	"github.com/johnquangdev/meeting-assistant/pkg/jwt"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize Echo instance
	e := echo.New()

	// Configure Echo
	e.HideBanner = true
	e.HidePort = false

	// Custom logger format
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${time_rfc3339} | ${status} | ${method} ${uri} | ${latency_human}\n",
	}))

	// Recover from panics
	e.Use(middleware.Recover())

	// CORS middleware
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: cfg.Server.AllowedOrigins,
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	// Initialize dependencies
	log.Println("⚙️  Initializing OAuth components...")

	// Initialize Database
	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.CloseDB(db)

	// Run migrations
	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize Redis
	redisClient, err := cache.NewRedisClient(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)

	// Initialize OAuth provider
	googleProvider := oauth.NewGoogleProvider(
		cfg.OAuth.Google.ClientID,
		cfg.OAuth.Google.ClientSecret,
		cfg.OAuth.Google.RedirectURL,
	)

	// Initialize state manager with Redis for CSRF protection
	stateManager := oauth.NewStateManager(redisClient)

	// Initialize JWT manager
	jwtManager := jwt.NewManager(
		cfg.JWT.AccessSecret,
		cfg.JWT.RefreshSecret,
		cfg.JWT.AccessExpiry,
		cfg.JWT.RefreshExpiry,
	)

	// Initialize OAuth service with real repositories
	oauthService := auth.NewOAuthService(
		userRepo,
		sessionRepo,
		googleProvider,
		stateManager,
		jwtManager,
	)

	// Initialize auth handler
	authHandler := handler.NewAuth(oauthService)
	log.Println("✅ Auth handler initialized successfully")

	// Setup router with handlers
	router := handler.NewRouter(cfg, authHandler)
	router.Setup(e)

	// Start server
	go func() {
		addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
		log.Printf("🚀 Starting server on %s", addr)
		log.Printf("📝 Environment: %s", cfg.Server.Environment)
		log.Printf("🔗 Health check: http://%s/health", addr)

		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Server.ShutdownTimeout)*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Fatalf("❌ Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server stopped gracefully")
}
