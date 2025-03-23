// Package batcher provides in-memory batching and delivery logic.
package batcher

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"benzinga-webhook/internal/config"
	"benzinga-webhook/internal/model"

	"go.uber.org/zap"
)

// Batcher defines the interface for adding entries and controlling lifecycle.
type Batcher interface {
	Add(entry model.LogEntry)
	Start()
	Stop()
}

// Batcher holds buffered log entries and manages periodic flushing.
type batcher struct {
	log     *zap.Logger
	cfg     *config.Config
	entries []model.LogEntry
	mu      sync.Mutex
	ticker  *time.Ticker
	quit    chan struct{}
}

// New initializes a new Batcher instance.
func New(cfg *config.Config, logger *zap.Logger) Batcher {
	return &batcher{
		log:    logger,
		cfg:    cfg,
		quit:   make(chan struct{}),
		ticker: time.NewTicker(cfg.BatchInterval),
	}
}

// Add appends a log entry to the batch buffer.
func (b *batcher) Add(entry model.LogEntry) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries = append(b.entries, entry)
	if len(b.entries) >= b.cfg.BatchSize {
		go b.flush()
	}
}

// Start runs the periodic flush ticker.
func (b *batcher) Start() {
	for {
		select {
		case <-b.ticker.C:
			b.flush()
		case <-b.quit:
			b.flush()
			b.ticker.Stop()
			return
		}
	}
}

// Stop signals the batcher to flush and shutdown.
func (b *batcher) Stop() {
	close(b.quit)
}

func (b *batcher) flush() {
	b.mu.Lock()
	if len(b.entries) == 0 {
		b.mu.Unlock()
		return
	}
	batch := b.entries
	b.entries = nil
	b.mu.Unlock()

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
		b.log.Warn("POST failed", zap.Int("attempt", i), zap.Error(err))
		time.Sleep(2 * time.Second)
	}

	duration := time.Since(start)
	if err != nil || resp.StatusCode >= 300 {
		b.log.Error("batch failed after 3 attempts", zap.Int("size", len(batch)), zap.Error(err))
		return
	}

	b.log.Info("batch sent successfully",
		zap.Int("size", len(batch)),
		zap.Int("status", resp.StatusCode),
		zap.Duration("duration", duration))
}
