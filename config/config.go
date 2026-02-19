package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
	DatabaseURL   string
}

func Load() *Config {
	// Load .env file if it exists (ignore errors if file doesn't exist)
	_ = godotenv.Load()

	// Get configuration from environment variables with fallback defaults
	port := getEnv("PORT", ":8080")
	accessSecret := getEnv("ACCESS_SECRET", "your-access-secret")
	refreshSecret := getEnv("REFRESH_SECRET", "your-refresh-secret")
	databaseURL := getEnv("DATABASE_URL", "postgres://user:password@localhost:5432/rubxy?sslmode=disable")

	// Warn if using default secrets in production
	if accessSecret == "your-access-secret" || refreshSecret == "your-refresh-secret" {
		log.Println("WARNING: Using default secrets. Please set ACCESS_SECRET and REFRESH_SECRET environment variables!")
	}

	return &Config{
		Port:          port,
		AccessSecret:  accessSecret,
		RefreshSecret: refreshSecret,
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    7 * 24 * time.Hour,
		DatabaseURL:   databaseURL,
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
