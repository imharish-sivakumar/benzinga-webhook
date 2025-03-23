package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

func TestNewLogger_Development(t *testing.T) {
	log := New("development")
	assert.NotNil(t, log)

	// Check that the logger uses development config (has DebugLevel enabled)
	core := log.Core()
	assert.True(t, core.Enabled(zapcore.DebugLevel), "development logger should allow debug level")
}

func TestNewLogger_Production(t *testing.T) {
	log := New("production")
	assert.NotNil(t, log)

	// Check that production logger disables debug logging
	core := log.Core()
	assert.False(t, core.Enabled(zapcore.DebugLevel), "production logger should not allow debug level")
}
