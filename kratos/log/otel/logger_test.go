package otel

import (
	"context"
	"testing"

	kratoslog "github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
)

type recordingLoggerProvider struct {
	embedded.LoggerProvider

	name   string
	config log.LoggerConfig
	logger *recordingLogger
}

func (p *recordingLoggerProvider) Logger(name string, opts ...log.LoggerOption) log.Logger {
	p.name = name
	p.config = log.NewLoggerConfig(opts...)
	p.logger = &recordingLogger{}
	return p.logger
}

type recordingLogger struct {
	embedded.Logger

	ctx          context.Context
	record       log.Record
	emitted      bool
	enabled      bool
	enabledCtx   context.Context
	enabledParam log.EnabledParameters
}

func (l *recordingLogger) Emit(ctx context.Context, record log.Record) {
	l.ctx = ctx
	l.record = record.Clone()
	l.emitted = true
}

func (l *recordingLogger) Enabled(ctx context.Context, param log.EnabledParameters) bool {
	l.enabledCtx = ctx
	l.enabledParam = param
	return l.enabled
}

func TestNewLoggerUsesScopeNameAndVersion(t *testing.T) {
	provider := &recordingLoggerProvider{}

	logger := NewLogger(WithLoggerProvider(provider))

	require.NotNil(t, logger)
	assert.Equal(t, scopeName, provider.name)
	assert.Equal(t, Version(), provider.config.InstrumentationVersion())
}

func TestLoggerLogEmitsRecord(t *testing.T) {
	provider := &recordingLoggerProvider{}
	logger := NewLogger(WithLoggerProvider(provider))
	provider.logger.enabled = true

	ctx := context.WithValue(context.Background(), struct{}{}, "value")
	err := logger.Log(
		kratoslog.LevelInfo,
		kratoslog.DefaultMessageKey, "hello",
		"request_id", "req-1",
		"context", ctx,
	)

	require.NoError(t, err)
	require.NotNil(t, provider.logger)
	require.True(t, provider.logger.emitted)
	assert.Same(t, ctx, provider.logger.enabledCtx)
	assert.Equal(t, log.SeverityInfo, provider.logger.enabledParam.Severity)
	assert.Same(t, ctx, provider.logger.ctx)
	assert.Equal(t, log.SeverityInfo, provider.logger.record.Severity())
	assert.Equal(t, kratoslog.LevelInfo.String(), provider.logger.record.SeverityText())
	assert.Equal(t, log.StringValue("hello"), provider.logger.record.Body())
	assert.Equal(t, 1, provider.logger.record.AttributesLen())

	var attrs []log.KeyValue
	provider.logger.record.WalkAttributes(func(attr log.KeyValue) bool {
		attrs = append(attrs, attr)
		return true
	})
	require.Len(t, attrs, 1)
	assert.Equal(t, "request_id", attrs[0].Key)
	assert.Equal(t, log.StringValue("req-1"), attrs[0].Value)
}

func TestLoggerLogSkipsEmitWhenDisabled(t *testing.T) {
	provider := &recordingLoggerProvider{}
	logger := NewLogger(WithLoggerProvider(provider))
	provider.logger.enabled = false

	ctx := context.WithValue(context.Background(), struct{}{}, "value")
	err := logger.Log(
		kratoslog.LevelError,
		kratoslog.DefaultMessageKey, "ignored",
		"context", ctx,
	)

	require.NoError(t, err)
	assert.Same(t, ctx, provider.logger.enabledCtx)
	assert.Equal(t, log.SeverityError, provider.logger.enabledParam.Severity)
	assert.False(t, provider.logger.emitted)
}
