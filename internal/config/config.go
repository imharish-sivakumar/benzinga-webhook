// Package config handles application configuration via environment variables.
package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

// Config holds all configurable values for the app.
type Config struct {
	Env           string
	BatchSize     int
	BatchInterval time.Duration
	PostEndpoint  string
}

// Load reads environment variables and populates a Config struct.
func Load() *Config {
	batchSize, err := strconv.Atoi(getEnv("BATCH_SIZE", "5"))
	if err != nil {
		log.Panicf("Invalid BATCH_SIZE: %v", err)
	}

	interval, err := time.ParseDuration(getEnv("BATCH_INTERVAL", "10s"))
	if err != nil {
		log.Panicf("Invalid BATCH_INTERVAL: %v", err)
	}

	return &Config{
		Env:           getEnv("ENV", "development"),
		BatchSize:     batchSize,
		BatchInterval: interval,
		PostEndpoint:  getEnv("POST_ENDPOINT", "http://localhost:9000"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
