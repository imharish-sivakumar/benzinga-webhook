package main

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRun_StartsAndShutsDown(t *testing.T) {
	t.Setenv("ENV", "development")
	t.Setenv("BATCH_SIZE", "10")
	t.Setenv("BATCH_INTERVAL", "1s")
	t.Setenv("POST_ENDPOINT", "http://localhost:9999") // dummy endpoint

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := Run(ctx)
	assert.NoError(t, err)
}

func TestMain_GracefulExit(t *testing.T) {
	// Set environment variables so config.Load() doesn't panic
	t.Setenv("ENV", "test")
	t.Setenv("BATCH_SIZE", "5")
	t.Setenv("BATCH_INTERVAL", "1s")
	t.Setenv("POST_ENDPOINT", "http://localhost:12345") // dummy endpoint

	// Run main in a goroutine (this will block waiting for signal)
	go func() {
		main()
	}()

	// Give time for main to start
	time.Sleep(500 * time.Millisecond)

	// Send SIGINT to simulate Ctrl+C
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("unable to find process: %v", err)
	}
	_ = p.Signal(syscall.SIGINT)

	// Wait for graceful shutdown
	time.Sleep(1 * time.Second)
}
