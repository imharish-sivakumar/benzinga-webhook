// Package handler contains HTTP handlers for the webhook API.
package handler

import (
	"encoding/json"
	"net/http"
	"regexp"

	"benzinga-webhook/internal/apperror"
	"benzinga-webhook/internal/batcher"
	"benzinga-webhook/internal/model"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// PhoneValidator validates phone numbers using a custom pattern format (e.g., 123-4567-891).
var PhoneValidator = func(fl validator.FieldLevel) bool {
	pattern := `^\d{3}-\d{4}-\d{3}$`
	matched, _ := regexp.MatchString(pattern, fl.Field().String())
	return matched
}

// Handler wraps HTTP handlers with logger and batcher.
type Handler struct {
	log      *zap.Logger
	batch    batcher.Batcher
	validate *validator.Validate
}

// New creates a new Handler instance.
func New(log *zap.Logger, b batcher.Batcher, v *validator.Validate) *Handler {
	return &Handler{log: log, batch: b, validate: v}
}

// Healthz is a simple health check endpoint.
func (h *Handler) Healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

// LogPayload receives and processes JSON payloads.
func (h *Handler) LogPayload(w http.ResponseWriter, r *http.Request) {
	var entry model.LogEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		h.log.Error("failed to decode json", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid request payload",
		})
		return
	}

	if err := h.validate.Struct(entry); err != nil {
		h.log.Warn("validation failed", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		validationError := apperror.CustomValidationError(err)
		if err := json.NewEncoder(w).Encode(validationError); err != nil {
			h.log.Error("unable to write response stream", zap.Error(err))
			return
		}
		return
	}

	h.batch.Add(entry)
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "Ok",
	})
}
