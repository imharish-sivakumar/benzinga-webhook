package batcher

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"benzinga-webhook/internal/config"
	"benzinga-webhook/internal/model"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

type captureServer struct {
	t         *testing.T
	lock      sync.Mutex
	calls     int
	lastBatch []model.LogEntry
}

func (s *captureServer) handler(w http.ResponseWriter, r *http.Request) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.calls++
	body, err := io.ReadAll(r.Body)
	assert.NoError(s.t, err)

	var entries []model.LogEntry
	err = json.Unmarshal(body, &entries)
	assert.NoError(s.t, err)

	s.lastBatch = entries

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func sampleEntry() model.LogEntry {
	return model.LogEntry{
		UserID:    1,
		Total:     9.99,
		Title:     "Test Payload",
		Completed: false,
		Meta: model.Meta{
			Logins: []model.Login{{Time: "2020-08-08T01:52:50Z", IP: "127.0.0.1"}},
			PhoneNumbers: model.PhoneNumbers{
				Home:   "555-1212-123",
				Mobile: "555-3434-999",
			},
		},
	}
}

func TestBatcher_FlushBySize(t *testing.T) {
	server := &captureServer{t: t}
	ts := httptest.NewServer(http.HandlerFunc(server.handler))
	defer ts.Close()

	cfg := &config.Config{
		BatchSize:     2,
		BatchInterval: 10 * time.Second,
		PostEndpoint:  ts.URL,
	}

	b := New(cfg, zaptest.NewLogger(t))
	go b.Start()
	defer b.Stop()

	b.Add(sampleEntry())
	b.Add(sampleEntry())
	time.Sleep(1 * time.Second)

	server.lock.Lock()
	defer server.lock.Unlock()
	assert.Equal(t, 1, server.calls)
	assert.Len(t, server.lastBatch, 2)
}

func TestBatcher_FlushByInterval(t *testing.T) {
	server := &captureServer{t: t}
	ts := httptest.NewServer(http.HandlerFunc(server.handler))
	defer ts.Close()

	cfg := &config.Config{
		BatchSize:     100,
		BatchInterval: 1 * time.Second,
		PostEndpoint:  ts.URL,
	}

	b := New(cfg, zaptest.NewLogger(t))
	go b.Start()
	defer b.Stop()

	b.Add(sampleEntry())
	time.Sleep(2 * time.Second)

	server.lock.Lock()
	defer server.lock.Unlock()
	assert.Equal(t, 1, server.calls)
	assert.Len(t, server.lastBatch, 1)
}

func TestBatcher_FlushEmpty(t *testing.T) {
	cfg := &config.Config{
		BatchSize:     10,
		BatchInterval: 1 * time.Second,
		PostEndpoint:  "http://localhost", // unreachable
	}

	b := batcher{log: zaptest.NewLogger(t), cfg: cfg, quit: make(chan struct{}), ticker: time.NewTicker(cfg.BatchInterval)}
	b.flush()
}

func TestBatcher_FlushFailure(t *testing.T) {
	// simulate failure by using an invalid endpoint
	cfg := &config.Config{
		BatchSize:     1,
		BatchInterval: 1 * time.Second,
		PostEndpoint:  "http://invalid-host.local",
	}

	b := New(cfg, zaptest.NewLogger(t)).(*batcher)
	b.entries = []model.LogEntry{sampleEntry()}

	b.flush()
}
