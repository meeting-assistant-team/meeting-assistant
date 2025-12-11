package config

import (
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// Config holds application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	OAuth    OAuthConfig
	JWT      JWTConfig
	Storage  StorageConfig
	LiveKit  LiveKitConfig
	Assembly AssemblyAIConfig
	Groq     GroqConfig
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port            string   `envconfig:"PORT"`
	Host            string   `envconfig:"HOST"`
	Environment     string   `envconfig:"ENVIRONMENT"`
	AllowedOrigins  []string `envconfig:"ALLOWED_ORIGINS" split_words:"true"`
	ShutdownTimeout int      `envconfig:"SHUTDOWN_TIMEOUT" default:"15"`
	// FrontendURL is the URL of the frontend application (used for redirects after login)
	FrontendURL string `envconfig:"FRONTEND_URL" default:"https://meeting-assistant.infoquang.id.vn"`
	// Cookie settings for authentication cookies
	CookieDomain   string `envconfig:"COOKIE_DOMAIN"`
	CookiePath     string `envconfig:"COOKIE_PATH" default:"/"`
	CookieSecure   bool   `envconfig:"COOKIE_SECURE" default:"true"`
	CookieSameSite string `envconfig:"COOKIE_SAME_SITE" default:"lax"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string `envconfig:"DB_HOST"`
	Port     string `envconfig:"DB_PORT"`
	User     string `envconfig:"DB_USER"`
	Password string `envconfig:"DB_PASSWORD"`
	Name     string `envconfig:"DB_NAME"`
	SSLMode  string `envconfig:"DB_SSLMODE"`
	MaxConns int    `envconfig:"DB_MAX_CONNS"`
	MinConns int    `envconfig:"DB_MIN_CONNS"`
}

// OAuthConfig holds OAuth configuration
type OAuthConfig struct {
	Google GoogleOAuthConfig
}

// GoogleOAuthConfig holds Google OAuth configuration
type GoogleOAuthConfig struct {
	ClientID     string `envconfig:"GOOGLE_CLIENT_ID"`
	ClientSecret string `envconfig:"GOOGLE_CLIENT_SECRET"`
	RedirectURL  string `envconfig:"GOOGLE_REDIRECT_URL"`
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	AccessSecret  string        `envconfig:"JWT_ACCESS_SECRET"`
	RefreshSecret string        `envconfig:"JWT_REFRESH_SECRET"`
	AccessExpiry  time.Duration `envconfig:"JWT_ACCESS_TOKEN_EXPIRE"`
	RefreshExpiry time.Duration `envconfig:"JWT_REFRESH_TOKEN_EXPIRE"`
}

// StorageConfig holds storage configuration
type StorageConfig struct {
	Type            string // "minio" or "s3"
	Endpoint        string `envconfig:"MINIO_ENDPOINT"`
	AccessKeyID     string `envconfig:"MINIO_ACCESS_KEY"`
	SecretAccessKey string `envconfig:"MINIO_SECRET_KEY"`
	BucketName      string `envconfig:"MINIO_BUCKET_NAME"`
	UseSSL          bool   `envconfig:"MINIO_USE_SSL"`
}

// LiveKitConfig holds LiveKit configuration
type LiveKitConfig struct {
	URL           string `envconfig:"LIVEKIT_URL" default:"ws://localhost:7880"`
	APIKey        string `envconfig:"LIVEKIT_API_KEY" default:"devkey"`
	APISecret     string `envconfig:"LIVEKIT_API_SECRET" default:"secret"`
	WebhookSecret string `envconfig:"LIVEKIT_WEBHOOK_SECRET"`           // Secret for validating webhooks from LiveKit
	WebhookURL    string `envconfig:"LIVEKIT_WEBHOOK_URL"`              // Webhook URL for LiveKit to call back (must be publicly accessible)
	UseMock       bool   `envconfig:"LIVEKIT_USE_MOCK" default:"false"` // Use mock mode for testing without real LiveKit server
}

// AssemblyAIConfig holds AssemblyAI related configuration
type AssemblyAIConfig struct {
	APIKey         string `envconfig:"ASSEMBLYAI_API_KEY"`
	WebhookSecret  string `envconfig:"ASSEMBLYAI_WEBHOOK_SECRET"`
	WebhookBaseURL string `envconfig:"ASSEMBLYAI_WEBHOOK_BASE_URL"`
}

// GroqConfig holds Groq API config
type GroqConfig struct {
	APIKey  string `envconfig:"GROQ_API_KEY"`
	BaseURL string `envconfig:"GROQ_API_URL"`
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	config := &Config{}
	// Load .env file if exists (ignore error if file doesn't exist)
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found, using environment variables or defaults")
	}
	// Process environment variables into config struct
	err := envconfig.Process("", config)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return config, nil
}

// GetDatabaseDSN returns the database connection string
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

// GetS3Endpoint returns the S3/MinIO endpoint with protocol
func (s *StorageConfig) GetS3Endpoint() string {
	protocol := "http://"
	if s.UseSSL {
		protocol = "https://"
	}

	// Check if endpoint already has protocol
	if len(s.Endpoint) > 7 && (s.Endpoint[:7] == "http://" || s.Endpoint[:8] == "https://") {
		return s.Endpoint
	}

	return protocol + s.Endpoint
}
