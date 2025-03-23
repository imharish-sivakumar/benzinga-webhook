package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoad_Defaults(t *testing.T) {
	os.Clearenv()

	cfg := Load()

	assert.Equal(t, "development", cfg.Env)
	assert.Equal(t, 5, cfg.BatchSize)
	assert.Equal(t, 10*time.Second, cfg.BatchInterval)
	assert.Equal(t, "http://localhost:9000", cfg.PostEndpoint)
}

func TestLoad_CustomEnv(t *testing.T) {
	_ = os.Setenv("ENV", "production")
	_ = os.Setenv("BATCH_SIZE", "15")
	_ = os.Setenv("BATCH_INTERVAL", "30s")
	_ = os.Setenv("POST_ENDPOINT", "https://example.com/hook")

	cfg := Load()

	assert.Equal(t, "production", cfg.Env)
	assert.Equal(t, 15, cfg.BatchSize)
	assert.Equal(t, 30*time.Second, cfg.BatchInterval)
	assert.Equal(t, "https://example.com/hook", cfg.PostEndpoint)
}

func TestLoad_InvalidBatchSize(t *testing.T) {
	_ = os.Setenv("BATCH_SIZE", "invalid")
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic due to invalid BATCH_SIZE")
		}
	}()
	Load()
}

func TestLoad_InvalidInterval(t *testing.T) {
	_ = os.Setenv("BATCH_SIZE", "5")
	_ = os.Setenv("BATCH_INTERVAL", "invalid-duration")
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic due to invalid BATCH_INTERVAL")
		}
	}()
	Load()
}
