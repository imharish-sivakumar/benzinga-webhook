// Package batcher provides in-memory batching and delivery logic.
package batcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"benzinga-webhook/internal/config"
	"benzinga-webhook/internal/model"

	"go.uber.org/zap"
)

var exitFunc = os.Exit

// Batcher defines the interface for adding entries and controlling lifecycle.
type Batcher interface {
	Add(entry model.LogEntry)
	Start()
	Stop()
}

// batcher holds buffered log entries and manages periodic flushing.
type batcher struct {
	log     *zap.Logger
	cfg     *config.Config
	entries chan model.LogEntry
	quit    chan struct{}
}

// New initializes a new Batcher instance.
func New(cfg *config.Config, logger *zap.Logger) Batcher {
	return &batcher{
		log:     logger,
		cfg:     cfg,
		entries: make(chan model.LogEntry, 1000),
		quit:    make(chan struct{}),
	}
}

// Add queues a log entry into the batch channel.
func (b *batcher) Add(entry model.LogEntry) {
	select {
	case b.entries <- entry:
		// successfully added
	default:
		b.log.Warn("entry channel full, dropping entry")
	}
}

// Start runs the periodic flush ticker and processes the batch channel.
func (b *batcher) Start() {
	buffer := make([]model.LogEntry, 0, b.cfg.BatchSize)
	ticker := time.NewTicker(b.cfg.BatchInterval)
	defer ticker.Stop()

	for {
		select {
		case entry := <-b.entries:
			buffer = append(buffer, entry)
			if len(buffer) >= b.cfg.BatchSize {
				b.flush(buffer)
				buffer = nil
			}
		case <-ticker.C:
			if len(buffer) > 0 {
				b.flush(buffer)
				buffer = nil
			}
		case <-b.quit:
			if len(buffer) > 0 {
				b.flush(buffer)
			}
			return
		}
	}
}

// Stop signals the batcher to flush and shutdown.
func (b *batcher) Stop() {
	close(b.quit)
}

func (b *batcher) flush(batch []model.LogEntry) {
	payload, err := json.Marshal(batch)
	if err != nil {
		b.log.Error("failed to marshal batch", zap.Error(err))
		return
	}

	start := time.Now()
	var resp *http.Response
	for i := 1; i <= 3; i++ {
		req, _ := http.NewRequest("POST", b.cfg.PostEndpoint, bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err = client.Do(req)
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			break
		}
		b.log.Warn("POST failed", zap.Int("attempt", i), zap.Any("error", err), zap.Int("statusCode", resp.StatusCode))
		time.Sleep(2 * time.Second)
	}
	fmt.Println("coming here")
	duration := time.Since(start)
	if err != nil || resp.StatusCode >= 300 {
		b.log.Error("batch failed after 3 attempts", zap.Int("size", len(batch)), zap.Error(err))
		exitFunc(1)
		return
	}

	b.log.Info("batch sent successfully",
		zap.Int("size", len(batch)),
		zap.Int("status", resp.StatusCode),
		zap.Duration("duration", duration))
}
