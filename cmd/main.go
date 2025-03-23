// Package main provides entry point for the log application.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"

	"benzinga-webhook/internal/batcher"
	"benzinga-webhook/internal/config"
	"benzinga-webhook/internal/handler"
	"benzinga-webhook/internal/logger"
)

// Run is the testable entrypoint for the application.
func Run(ctx context.Context) error {
	cfg := config.Load()
	log := logger.New(cfg.Env)
	log.Info("Starting Benzinga Webhook Receiver")

	r := chi.NewRouter()
	batch := batcher.New(cfg, log)
	validate := validator.New()
	_ = validate.RegisterValidation("phoneformat", handler.PhoneValidator)

	h := handler.New(log, batch, validate)
	r.Get("/healthz", h.Healthz)
	r.Post("/log", h.LogPayload)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go batch.Start()
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	<-ctx.Done()

	log.Info("Shutting down server")
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctxShutdown)
	batch.Stop()
	return nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := Run(ctx); err != nil {
		os.Exit(1)
	}
}
