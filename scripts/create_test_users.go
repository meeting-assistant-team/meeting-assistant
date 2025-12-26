package main

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/johnquangdev/meeting-assistant/internal/domain/entities"
	"github.com/johnquangdev/meeting-assistant/internal/infrastructure/database"
	"github.com/johnquangdev/meeting-assistant/pkg/config"
	pkgjwt "github.com/johnquangdev/meeting-assistant/pkg/jwt"
)

func main() {
	log.Println("🚀 Starting test users creation...")

	// Load configuration from .env
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	log.Println("📦 Connecting to database...")
	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.CloseDB(db)

	log.Println("✅ Database connected successfully\n")

	// Initialize JWT manager
	jwtManager := pkgjwt.NewManager(
		cfg.JWT.AccessSecret,
		cfg.JWT.RefreshSecret,
		cfg.JWT.AccessExpiry,
		cfg.JWT.RefreshExpiry,
	)

	// Define test users
	testUsers := []struct {
		Email string
		Name  string
	}{
		{Email: "alice@test.local", Name: "Alice"},
		{Email: "bob@test.local", Name: "Bob"},
		{Email: "charlie@test.local", Name: "Charlie"},
		{Email: "diana@test.local", Name: "Diana"},
		{Email: "eve@test.local", Name: "Eve"},
	}

	log.Println("🗑️  Cleaning up existing test users...")
	// Delete existing sessions and users
	db.Where("user_id IN (SELECT id FROM users WHERE email LIKE ?)", "%@test.local").Delete(&entities.Session{})
	db.Where("email LIKE ?", "%@test.local").Delete(&entities.User{})

	log.Println("🔑 Creating test users and tokens...\n")

	// Create users and tokens
	for i, testUser := range testUsers {
		// Create user
		user := &entities.User{
			ID:              uuid.New(),
			Email:           testUser.Email,
			Name:            testUser.Name,
			Role:            entities.RoleParticipant,
			IsActive:        true,
			IsEmailVerified: true,
			Timezone:        "UTC",
			Language:        "en",
		}

		// Create new user
		if err := db.Create(user).Error; err != nil {
			log.Printf("❌ Failed to create user %s: %v", testUser.Email, err)
			continue
		}

		// Generate access token (with default expiry)
		accessToken, err := jwtManager.GenerateAccessToken(user.ID, user.Email, string(user.Role))
		if err != nil {
			log.Printf("❌ Failed to generate access token for %s: %v", testUser.Email, err)
			continue
		}

		// Generate access token for dev (with long expiry)
		devAccessToken, err := jwtManager.GenerateAccessTokenWithExpiry(user.ID, user.Email, string(user.Role), cfg.JWT.DevAccessExpiry)
		if err != nil {
			log.Printf("❌ Failed to generate dev access token for %s: %v", testUser.Email, err)
			continue
		}

		// Generate refresh token
		refreshToken, err := jwtManager.GenerateRefreshToken(user.ID)
		if err != nil {
			log.Printf("❌ Failed to generate refresh token for %s: %v", testUser.Email, err)
			continue
		}

		// Create session and save refresh token
		session := entities.NewSession(
			user.ID,
			refreshToken,
			time.Now().Add(cfg.JWT.RefreshExpiry),
		)

		if err := db.Create(session).Error; err != nil {
			log.Printf("❌ Failed to create session for %s: %v", testUser.Email, err)
			continue
		}

		// Print token infoDefault expiry: %v):\n", cfg.JWT.AccessExpiry)
		fmt.Printf("%s\n", accessToken)
		fmt.Printf("\n🔐 Dev Access Token (Long expiry: %v):\n", cfg.JWT.DevAccessExpiry)
		fmt.Printf("%s\n", devAccessToken)
		fmt.Printf("═══════════════════════════════════════════════════════\n")
		fmt.Printf("🟢 User %d: %s\n", i+1, testUser.Name)
		fmt.Printf("═══════════════════════════════════════════════════════════════\n")
		fmt.Printf("Email:        %s\n", user.Email)
		fmt.Printf("User ID:      %s\n", user.ID)
		fmt.Printf("Role:         %s\n", user.Role)
		fmt.Printf("\n📋 Access Token (Copy to Postman):\n")
		fmt.Printf("%s\n", accessToken)
		fmt.Printf("\n🔄 Refresh Token (Stored in DB):\n")
		fmt.Printf("%s\n", refreshToken)
		fmt.Printf("───────────────────────────────────────────────────────────────\n\n")
	}

	log.Println("✅ All test users created successfully!")
	log.Println("\n💡 Usage:")
	log.Println("   1. Copy the Access Token above")
	log.Println("   2. In Postman, set header: Authorization: Bearer <access_token>")
	log.Println("   3. Token expiry:", cfg.JWT.AccessExpiry)
	log.Println("\n🧹 To clean up test users, run: DELETE FROM users WHERE email LIKE '%@test.local'")
}
