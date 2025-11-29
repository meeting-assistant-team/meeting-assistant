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

	"go.uber.org/zap"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	pkgvalidator "github.com/johnquangdev/meeting-assistant/pkg/validator"

	"github.com/johnquangdev/meeting-assistant/internal/adapter/handler"
	"github.com/johnquangdev/meeting-assistant/internal/adapter/repository"
	"github.com/johnquangdev/meeting-assistant/internal/infrastructure/cache"
	"github.com/johnquangdev/meeting-assistant/internal/infrastructure/database"
	"github.com/johnquangdev/meeting-assistant/internal/infrastructure/external/livekit"
	"github.com/johnquangdev/meeting-assistant/internal/infrastructure/external/oauth"
	httpmw "github.com/johnquangdev/meeting-assistant/internal/infrastructure/http/middleware"
	aiuse "github.com/johnquangdev/meeting-assistant/internal/usecase/ai"
	"github.com/johnquangdev/meeting-assistant/internal/usecase/auth"
	"github.com/johnquangdev/meeting-assistant/internal/usecase/room"
	pkgai "github.com/johnquangdev/meeting-assistant/pkg/ai"
	"github.com/johnquangdev/meeting-assistant/pkg/config"
	"github.com/johnquangdev/meeting-assistant/pkg/jwt"
)

// @title           Meeting Assistant API
// @version         1.0
// @description     API for Meeting Assistant application with OAuth, room management, and LiveKit integration
// @termsOfService  https://api-meeting.infoquang.id.vn/terms

// @contact.name   API Support
// @contact.url    https://api-meeting.infoquang.id.vn/support
// @contact.email  support@infoquang.id.vn

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      api-meeting.infoquang.id.vn
// @BasePath  /v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize Echo instance
	e := echo.New()

	// Register validator for request validation
	e.Validator = pkgvalidator.New()

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
		AllowOrigins:     cfg.Server.AllowedOrigins,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "Set-Cookie", "Cookie"},
		AllowCredentials: true,
	}))

	// Initialize dependencies
	log.Println("🔧 Initializing dependencies...")

	// Initialize Database
	log.Println("📦 Connecting to database...")
	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.CloseDB(db)

	// Run AutoMigrate only when explicitly enabled in config.
	// Production deployments should manage schema via sql-migrate.
	if cfg.Database.AutoMigrate {
		if cfg.Server.Environment == "production" {
			log.Fatalf("AutoMigrate is enabled in production. Disable DB_AUTO_MIGRATE or manage schema with sql-migrate.")
		}
		log.Println("🔄 Running GORM AutoMigrate (development only) ...")
		if err := database.AutoMigrate(db); err != nil {
			log.Fatalf("Failed to run AutoMigrate: %v", err)
		}
	} else {
		log.Println("🔄 Skipping GORM AutoMigrate; use sql-migrate for schema migrations in CI/CD/production")
	}

	// Initialize Redis
	log.Println("📦 Connecting to Redis...")
	redisClient, err := cache.NewRedisClient(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize repositories
	log.Println("⚙️  Initializing repositories...")
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	roomRepo := repository.NewRoomRepository(db)
	participantRepo := repository.NewParticipantRepository(db)

	// Initialize AI repository and clients
	log.Println("🤖 Initializing AI components...")
	aiRepo := repository.NewAIRepository(db)
	asmClient := pkgai.NewAssemblyAIClient(&cfg.Assembly)
	groqClient := pkgai.NewGroqClient(&cfg.Groq)
	aiService := aiuse.NewAIService(aiRepo, asmClient, groqClient, cfg)
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()
	aiController := handler.NewAIController(aiService, logger)
	aiWebhookHandler := handler.NewAIWebhookHandler(aiService, cfg.Assembly.WebhookSecret, logger)

	// Initialize OAuth provider
	log.Println("🔐 Initializing OAuth provider...")
	googleProvider := oauth.NewGoogleProvider(
		cfg.OAuth.Google.ClientID,
		cfg.OAuth.Google.ClientSecret,
		cfg.OAuth.Google.RedirectURL,
	)

	// Initialize state manager with Redis for CSRF protection
	log.Println("🔒 Initializing state manager...")
	stateManager := oauth.NewStateManager(redisClient)

	// Initialize JWT manager
	log.Println("🔑 Initializing JWT manager...")
	jwtManager := jwt.NewManager(
		cfg.JWT.AccessSecret,
		cfg.JWT.RefreshSecret,
		cfg.JWT.AccessExpiry,
		cfg.JWT.RefreshExpiry,
	)

	// Initialize OAuth service with real repositories
	log.Println("✨ Initializing OAuth service...")
	oauthService := auth.NewOAuthService(
		userRepo,
		sessionRepo,
		redisClient,
		googleProvider,
		stateManager,
		jwtManager,
	)

	// Initialize auth handler
	log.Println("🚀 Initializing auth handler...")
	authHandler := handler.NewAuth(oauthService, logger, cfg)
	log.Println("✅ Auth handler initialized successfully")

	// Initialize LiveKit client
	log.Println("🎥 Initializing LiveKit client...")
	livekitClient := livekit.NewClient(
		cfg.LiveKit.URL,
		cfg.LiveKit.APIKey,
		cfg.LiveKit.APISecret,
		cfg.LiveKit.UseMock,
	)
	if cfg.LiveKit.UseMock {
		log.Println("⚠️  LiveKit running in MOCK mode (no real server needed)")
	} else {
		log.Printf("✅ LiveKit connected to: %s", cfg.LiveKit.URL)
	}

	// Initialize room service
	log.Println("🏠 Initializing room service...")
	roomService := room.NewRoomService(roomRepo, participantRepo, livekitClient, cfg.LiveKit.URL)

	// Initialize room handler
	log.Println("🚪 Initializing room handler...")
	roomHandler := handler.NewRoomHandler(roomService, logger)
	log.Println("✅ Room handler initialized successfully")

	// Initialize webhook handler (for LiveKit webhooks)
	log.Println("🪝 Initializing webhook handler...")
	webhookHandler := handler.NewWebhookHandler(roomService, cfg.LiveKit.WebhookSecret, logger)
	log.Println("✅ Webhook handler initialized successfully")

	// Setup router with handlers
	log.Println("🛣️  Setting up routes...")

	// Create Echo auth middleware from existing OAuth service
	authEchoMW := httpmw.EchoAuth(oauthService)

	router := handler.NewRouter(cfg, authHandler, roomHandler, webhookHandler, aiWebhookHandler, aiController, authEchoMW)
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
