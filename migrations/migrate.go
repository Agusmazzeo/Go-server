package main

import (
	"fmt"
	"log"
	"os"
	"server/src/config"

	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load the appropriate config based on the environment

	cfg, err := config.LoadConfig("./settings", os.Getenv("ENV"))
	if err != nil {
		log.Fatalf("Error loading config for environment: %v", err)
	}

	dsn := cfg.Databases.SQL.ConnectionString
	if dsn == "" {
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
			cfg.Databases.SQL.Host,
			cfg.Databases.SQL.Username,
			cfg.Databases.SQL.Password,
			cfg.Databases.SQL.Database,
			cfg.Databases.SQL.Port)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get SQL DB from GORM DB: %v", err)
	}

	// Apply migrations
	if err := goose.Up(sqlDB, "./migrations"); err != nil {
		log.Fatalf("Failed to apply migrations: %v", err)
	}

	log.Println("Database migration completed successfully")
}
