# Shared Packages (pkg/)

**Reusable packages that can be imported by any layer**

## Purpose
Contains common utilities and packages that are shared across different layers of the application. These should be generic enough to be extracted into separate libraries.

## Structure

### `config/`
Configuration management.

**Files:**
- `config.go` - Configuration loader and validator

**Example:**
```go
// config/config.go
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	JWT       JWTConfig
	OAuth     OAuthConfig
	LiveKit   LiveKitConfig
	OpenAI    OpenAIConfig
	Minio     MinioConfig
	CORS      CORSConfig
	RateLimit RateLimitConfig
}

type ServerConfig struct {
	Port string
	Host string
	Env  string
}

type DatabaseConfig struct {
	Host         string
	Port         int
	User         string
	Password     string
	DBName       string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
}

type RedisConfig struct {
	URL      string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret         string
	AccessExpiry   time.Duration
	RefreshExpiry  time.Duration
}

type OAuthConfig struct {
	Google GoogleOAuthConfig
}

type GoogleOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type LiveKitConfig struct {
	URL       string
	APIKey    string
	APISecret string
}

type OpenAIConfig struct {
	APIKey string
	Model  string
}

type MinioConfig struct {
	Endpoint   string
	AccessKey  string
	SecretKey  string
	BucketName string
	UseSSL     bool
}

type CORSConfig struct {
	AllowedOrigins []string
}

type RateLimitConfig struct {
	Requests int
	Window   time.Duration
}

func Load() (*Config, error) {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found, using environment variables")
	}

	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Host: getEnv("SERVER_HOST", "localhost"),
			Env:  getEnv("ENV", "development"),
		},
		Database: DatabaseConfig{
			Host:         getEnv("DB_HOST", "localhost"),
			Port:         getEnvAsInt("DB_PORT", 5432),
			User:         getEnv("DB_USER", "postgres"),
			Password:     getEnv("DB_PASSWORD", "postgres"),
			DBName:       getEnv("DB_NAME", "meeting_assistant"),
			SSLMode:      getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns: getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", "redis://localhost:6379/0"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:        requireEnv("JWT_SECRET"),
			AccessExpiry:  parseDuration(getEnv("JWT_ACCESS_EXPIRY", "15m")),
			RefreshExpiry: parseDuration(getEnv("JWT_REFRESH_EXPIRY", "168h")),
		},
		OAuth: OAuthConfig{
			Google: GoogleOAuthConfig{
				ClientID:     requireEnv("GOOGLE_CLIENT_ID"),
				ClientSecret: requireEnv("GOOGLE_CLIENT_SECRET"),
				RedirectURL:  requireEnv("GOOGLE_REDIRECT_URL"),
			},
		},
		LiveKit: LiveKitConfig{
			URL:       requireEnv("LIVEKIT_URL"),
			APIKey:    requireEnv("LIVEKIT_API_KEY"),
			APISecret: requireEnv("LIVEKIT_API_SECRET"),
		},
		OpenAI: OpenAIConfig{
			APIKey: requireEnv("OPENAI_API_KEY"),
			Model:  getEnv("OPENAI_MODEL", "gpt-4"),
		},
		Minio: MinioConfig{
			Endpoint:   getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey:  getEnv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey:  getEnv("MINIO_SECRET_KEY", "minioadmin"),
			BucketName: getEnv("MINIO_BUCKET_NAME", "meeting-recordings"),
			UseSSL:     getEnvAsBool("MINIO_USE_SSL", false),
		},
		CORS: CORSConfig{
			AllowedOrigins: getEnvAsSlice("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
		},
		RateLimit: RateLimitConfig{
			Requests: getEnvAsInt("RATE_LIMIT_REQUESTS", 100),
			Window:   parseDuration(getEnv("RATE_LIMIT_WINDOW", "1m")),
		},
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func requireEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("Required environment variable %s is not set", key))
	}
	return value
}

// Helper functions...
```

### `middleware/`
Reusable middleware components.

**Files:**
- `auth.go` - JWT authentication middleware
- `cors.go` - CORS middleware
- `rate_limit.go` - Rate limiting middleware
- `logger.go` - Request logging middleware

### `validator/`
Input validation utilities.

**Files:**
- `validator.go` - Custom validator setup

## Rules

✅ **DO:**
- Keep packages generic and reusable
- Avoid business logic
- Write comprehensive tests
- Document public APIs

❌ **DON'T:**
- Depend on internal packages
- Include framework-specific code (unless necessary)
- Mix business logic with utilities
