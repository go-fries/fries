package otel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log/global"
)

func TestNewConfigDefaultLoggerProvider(t *testing.T) {
	cfg := newConfig()

	require.NotNil(t, cfg.provider)
	assert.Equal(t, global.GetLoggerProvider(), cfg.provider)
	assert.Equal(t, Version(), cfg.version)
	assert.Empty(t, cfg.schemaURL)
	assert.Empty(t, cfg.attributes)
}

func TestWithLoggerProvider(t *testing.T) {
	provider := &recordingLoggerProvider{}

	cfg := newConfig(WithLoggerProvider(provider))

	assert.Same(t, provider, cfg.provider)
}

func TestConfigOptions(t *testing.T) {
	cfg := newConfig(
		WithVersion("1.2.3"),
		WithSchemaURL("https://example.com/schema"),
		WithAttributes(attribute.String("component", "logger")),
		WithAttributes(attribute.String("layer", "kratos")),
	)

	assert.Equal(t, "1.2.3", cfg.version)
	assert.Equal(t, "https://example.com/schema", cfg.schemaURL)
	assert.Equal(t, []attribute.KeyValue{
		attribute.String("component", "logger"),
		attribute.String("layer", "kratos"),
	}, cfg.attributes)
}

func TestConfigNewLogger(t *testing.T) {
	provider := &recordingLoggerProvider{}
	cfg := newConfig(
		WithLoggerProvider(provider),
		WithVersion("1.2.3"),
		WithSchemaURL("https://example.com/schema"),
		WithAttributes(attribute.String("component", "logger")),
	)

	logger := cfg.newLogger(scopeName)

	require.NotNil(t, logger)
	assert.Equal(t, scopeName, provider.name)
	assert.Equal(t, "1.2.3", provider.config.InstrumentationVersion())
	assert.Equal(t, "https://example.com/schema", provider.config.SchemaURL())
	attributes := provider.config.InstrumentationAttributes()
	value, ok := attributes.Value("component")
	require.True(t, ok)
	assert.Equal(t, attribute.StringValue("logger"), value)
}
