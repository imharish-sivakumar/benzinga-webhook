// Package logger provides structured logging with zap.
package logger

import "go.uber.org/zap"

// New creates a new zap.Logger depending on the environment.
func New(env string) *zap.Logger {
	if env == "production" {
		logger, _ := zap.NewProduction()
		return logger
	}
	logger, _ := zap.NewDevelopment()
	return logger
}
