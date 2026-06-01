package otel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/log/global"
)

func TestNewConfigDefaultLoggerProvider(t *testing.T) {
	cfg := newConfig()

	require.NotNil(t, cfg.provider)
	assert.Equal(t, global.GetLoggerProvider(), cfg.provider)
}

func TestWithLoggerProvider(t *testing.T) {
	provider := &recordingLoggerProvider{}

	cfg := newConfig(WithLoggerProvider(provider))

	assert.Same(t, provider, cfg.provider)
}

func TestConfigNewLogger(t *testing.T) {
	provider := &recordingLoggerProvider{}
	cfg := newConfig(WithLoggerProvider(provider))

	logger := cfg.newLogger(scopeName)

	require.NotNil(t, logger)
	assert.Equal(t, scopeName, provider.name)
	assert.Equal(t, Version(), provider.config.InstrumentationVersion())
}
