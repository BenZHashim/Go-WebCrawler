package config

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"log"
	"os"
	"time"
)

type Config struct {
	// DatabaseURL maps to env var DB_URL.
	// We mark it as required so the app fails fast if it's missing.
	DatabaseURL string `envconfig:"DB_URL" required:"true"`

	// Workers maps to WORKERS. Default to 10 if not set.
	Workers int `envconfig:"WORKERS" default:"10"`

	// StartURL maps to START_URL.
	StartURL string `envconfig:"START_URL" default:"https://www.hollywoodreporter.com"`

	// BatchSize maps to BATCH_SIZE.
	BatchSize int `envconfig:"BATCH_SIZE" default:"20"`

	// RateLimit maps to RATE_LIMIT. We can even parse durations directly!
	RateLimit time.Duration `envconfig:"RATE_LIMIT" default:"2s"`
}

// Load processes environment variables and populates the Config struct.
func Load() (*Config, error) {
	// 1. Try to load .env file (if it exists)
	// We don't panic here because in Production (Docker/K8s),
	// there often is no .env file (vars are injected directly).
	if err := godotenv.Load(); err != nil {
		// Only log if the file actually exists but failed to load.
		// If it's missing, we assume we're in Prod.
		if _, statErr := os.Stat(".env"); statErr == nil {
			log.Printf("Warning: .env file found but could not be loaded: %v", err)
		}
	}

	// 2. Process Environment Variables (System + Loaded from .env)
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
