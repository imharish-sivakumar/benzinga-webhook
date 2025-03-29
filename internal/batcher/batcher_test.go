package batcher

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"benzinga-webhook/internal/config"
	"benzinga-webhook/internal/model"

	"go.uber.org/zap/zaptest"
)

type mockServer struct {
	Requests [][]model.LogEntry
	Fail     bool
	FailResp bool
	Hits     int32
	Server   *httptest.Server
}

func newMockServer(fail, failResp bool, passAt int32) *mockServer {
	s := &mockServer{
		Fail:     fail,
		FailResp: failResp,
	}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&s.Hits, 1)
		if passAt == s.Hits {
			body, _ := io.ReadAll(r.Body)
			var logs []model.LogEntry
			_ = json.Unmarshal(body, &logs)
			s.Requests = append(s.Requests, logs)
			w.WriteHeader(http.StatusOK)
			return
		}
		if s.Fail {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if s.FailResp {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var logs []model.LogEntry
		_ = json.Unmarshal(body, &logs)
		s.Requests = append(s.Requests, logs)
		w.WriteHeader(http.StatusOK)
	}))
	return s
}

func TestBatcherFlushOnQuit(t *testing.T) {
	logger := zaptest.NewLogger(t)
	srv := newMockServer(false, false, 1)
	defer srv.Server.Close()

	cfg := &config.Config{
		BatchSize:     10,
		BatchInterval: 5 * time.Second,
		PostEndpoint:  srv.Server.URL,
	}

	b := New(cfg, logger)
	go b.Start()
	b.Add(model.LogEntry{UserID: 1, Total: 1.23, Title: "flush-on-quit"})
	time.Sleep(500 * time.Millisecond)
	b.Stop()

	time.Sleep(500 * time.Millisecond)
	if len(srv.Requests) != 1 {
		t.Errorf("expected 1 flush, got %d", len(srv.Requests))
	}
}

func TestBatcherRetries(t *testing.T) {
	logger := zaptest.NewLogger(t)
	srv := newMockServer(true, false, -1)
	defer srv.Server.Close()

	cfg := &config.Config{
		BatchSize:     1,
		BatchInterval: 1 * time.Second,
		PostEndpoint:  srv.Server.URL,
	}

	// intercept os.Exit
	exited := int32(0)
	savedExit := os.Exit
	exitFunc = func(code int) {
		atomic.StoreInt32(&exited, 1)
	}
	defer func() { exitFunc = savedExit }()

	b := New(cfg, logger)
	go b.Start()
	b.Add(model.LogEntry{UserID: 2, Total: 2.34, Title: "retry-fail"})
	time.Sleep(10 * time.Second)

	if atomic.LoadInt32(&exited) != 1 {
		t.Error("expected exit after retries fail")
	}
}

func TestBatcherFlushUsingTicker(t *testing.T) {
	logger := zaptest.NewLogger(t)
	srv := newMockServer(false, false, 1)
	defer srv.Server.Close()

	cfg := &config.Config{
		BatchSize:     3,
		BatchInterval: 2 * time.Second,
		PostEndpoint:  srv.Server.URL,
	}

	exited := int32(0)
	savedExit := os.Exit
	exitFunc = func(code int) {
		atomic.StoreInt32(&exited, 1)
	}
	defer func() { exitFunc = savedExit }()

	b := New(cfg, logger)
	go b.Start()
	b.Add(model.LogEntry{UserID: 3, Total: 3.45, Title: "bad-resp"})
	time.Sleep(5 * time.Second)

	if atomic.LoadInt32(&exited) != 0 {
		t.Error("expected non zero exit code")
	}
}

func TestBatcherSuccessAfterRetry(t *testing.T) {
	logger := zaptest.NewLogger(t)
	srv := newMockServer(false, true, 2)
	defer srv.Server.Close()

	cfg := &config.Config{
		BatchSize:     1,
		BatchInterval: 10 * time.Second,
		PostEndpoint:  srv.Server.URL,
	}

	exited := int32(0)
	savedExit := os.Exit
	exitFunc = func(code int) {
		atomic.StoreInt32(&exited, 1)
	}
	defer func() { exitFunc = savedExit }()

	b := New(cfg, logger)
	go b.Start()
	b.Add(model.LogEntry{UserID: 3, Total: 3.45, Title: "bad-resp"})
	time.Sleep(8 * time.Second)

	if atomic.LoadInt32(&exited) != 0 {
		t.Error("expected exit due to bad HTTP status code")
	}
}
