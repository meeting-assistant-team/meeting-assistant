package main

import (
	"log"
	"os"

	migrate "github.com/rubenv/sql-migrate"

	"github.com/johnquangdev/meeting-assistant/internal/infrastructure/database"
	"github.com/johnquangdev/meeting-assistant/pkg/config"
)

func mainn() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database using GORM
	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("✅ Database connected successfully")

	// Apply migrations
	log.Println("🔄 Applying migrations from migrations/ directory...")

	migrations := &migrate.FileMigrationSource{
		Dir: "migrations",
	}

	// Get the underlying SQL database connection from GORM
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database connection: %v", err)
	}

	n, err := migrate.Exec(sqlDB, "postgres", migrations, migrate.Up)
	if err != nil {
		log.Fatalf("Failed to apply migrations: %v", err)
	}

	log.Printf("✅ Successfully applied %d migration(s)!\n", n)
	os.Exit(0)
}
