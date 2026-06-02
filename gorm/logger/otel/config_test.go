package otel

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log/global"
	"gorm.io/gorm/logger"
)

func TestNewConfigDefaults(t *testing.T) {
	cfg := newConfig()

	require.NotNil(t, cfg.provider)
	assert.Equal(t, global.GetLoggerProvider(), cfg.provider)
	assert.Equal(t, Version(), cfg.version)
	assert.Empty(t, cfg.schemaURL)
	assert.Empty(t, cfg.attributes)
	assert.Equal(t, logger.Warn, cfg.level)
	assert.Equal(t, 200*time.Millisecond, cfg.slowThreshold)
	assert.False(t, cfg.ignoreRecordNotFoundError)
	assert.False(t, cfg.parameterizedQueries)
}

func TestConfigOptions(t *testing.T) {
	provider := &recordingLoggerProvider{}

	cfg := newConfig(
		WithLoggerProvider(provider),
		WithVersion("1.2.3"),
		WithSchemaURL("https://example.com/schema"),
		WithAttributes(attribute.String("component", "gorm")),
		WithAttributes(attribute.String("layer", "database")),
		WithLogLevel(logger.Info),
		WithSlowThreshold(time.Second),
		WithIgnoreRecordNotFoundError(true),
		WithParameterizedQueries(true),
	)

	assert.Same(t, provider, cfg.provider)
	assert.Equal(t, "1.2.3", cfg.version)
	assert.Equal(t, "https://example.com/schema", cfg.schemaURL)
	assert.Equal(t, []attribute.KeyValue{
		attribute.String("component", "gorm"),
		attribute.String("layer", "database"),
	}, cfg.attributes)
	assert.Equal(t, logger.Info, cfg.level)
	assert.Equal(t, time.Second, cfg.slowThreshold)
	assert.True(t, cfg.ignoreRecordNotFoundError)
	assert.True(t, cfg.parameterizedQueries)
}

func TestConfigSkipsEmptyScopeOptions(t *testing.T) {
	cfg := newConfig(
		WithLoggerProvider(nil),
		WithVersion(""),
		WithSchemaURL(""),
	)

	assert.Equal(t, global.GetLoggerProvider(), cfg.provider)
	assert.Equal(t, Version(), cfg.version)
	assert.Empty(t, cfg.schemaURL)
}

func TestConfigNewLogger(t *testing.T) {
	provider := &recordingLoggerProvider{}
	cfg := newConfig(
		WithLoggerProvider(provider),
		WithVersion("1.2.3"),
		WithSchemaURL("https://example.com/schema"),
		WithAttributes(attribute.String("component", "gorm")),
	)

	otelLogger := cfg.newLogger(scopeName)

	require.NotNil(t, otelLogger)
	assert.Equal(t, scopeName, provider.name)
	assert.Equal(t, "1.2.3", provider.config.InstrumentationVersion())
	assert.Equal(t, "https://example.com/schema", provider.config.SchemaURL())
	attributes := provider.config.InstrumentationAttributes()
	value, ok := attributes.Value("component")
	require.True(t, ok)
	assert.Equal(t, attribute.StringValue("gorm"), value)
}
