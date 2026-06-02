package otel

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
	gormlogger "gorm.io/gorm/logger"
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

func TestNew(t *testing.T) {
	provider := &recordingLoggerProvider{}

	logger := New(WithLoggerProvider(provider))

	require.NotNil(t, logger)
	assert.Equal(t, scopeName, provider.name)
	assert.Equal(t, gormlogger.Warn, logger.level)
	assert.Equal(t, 200*time.Millisecond, logger.slowThreshold)
	assert.False(t, logger.ignoreRecordNotFoundError)
	assert.False(t, logger.parameterizedQueries)
}

func TestLogModeReturnsCopy(t *testing.T) {
	logger := New(WithLoggerProvider(&recordingLoggerProvider{}), WithLogLevel(gormlogger.Warn))

	leveled := logger.LogMode(gormlogger.Info)

	require.IsType(t, &Logger{}, leveled)
	assert.Equal(t, gormlogger.Warn, logger.level)
	assert.Equal(t, gormlogger.Info, leveled.(*Logger).level)
}

func TestInfoEmitsRecord(t *testing.T) {
	provider := &recordingLoggerProvider{}
	logger := New(WithLoggerProvider(provider), WithLogLevel(gormlogger.Info))
	provider.logger.enabled = true

	ctx := t.Context()
	logger.Info(ctx, "hello %s", "gorm")

	require.True(t, provider.logger.emitted)
	assert.Same(t, ctx, provider.logger.enabledCtx)
	assert.Equal(t, log.SeverityInfo, provider.logger.enabledParam.Severity)
	assert.Same(t, ctx, provider.logger.ctx)
	assert.Equal(t, log.SeverityInfo, provider.logger.record.Severity())
	assert.Equal(t, "INFO", provider.logger.record.SeverityText())
	assert.Equal(t, log.StringValue("hello gorm"), provider.logger.record.Body())
}

func TestWarnAndErrorRespectLogLevel(t *testing.T) {
	provider := &recordingLoggerProvider{}
	logger := New(WithLoggerProvider(provider), WithLogLevel(gormlogger.Error))
	provider.logger.enabled = true

	logger.Warn(t.Context(), "ignored")
	assert.False(t, provider.logger.emitted)

	logger.Error(t.Context(), "failed: %s", "db")
	require.True(t, provider.logger.emitted)
	assert.Equal(t, log.SeverityError, provider.logger.record.Severity())
	assert.Equal(t, log.StringValue("failed: db"), provider.logger.record.Body())
}

func TestEmitSkipsWhenDisabled(t *testing.T) {
	provider := &recordingLoggerProvider{}
	logger := New(WithLoggerProvider(provider), WithLogLevel(gormlogger.Info))
	provider.logger.enabled = false

	logger.Info(t.Context(), "ignored")

	assert.False(t, provider.logger.emitted)
	assert.Equal(t, log.SeverityInfo, provider.logger.enabledParam.Severity)
}

func TestTraceError(t *testing.T) {
	provider := &recordingLoggerProvider{}
	logger := New(WithLoggerProvider(provider), WithLogLevel(gormlogger.Error))
	provider.logger.enabled = true
	err := errors.New("database failed")

	logger.Trace(t.Context(), time.Now().Add(-10*time.Millisecond), func() (string, int64) {
		return "select * from users", 3
	}, err)

	require.True(t, provider.logger.emitted)
	assert.Equal(t, log.SeverityError, provider.logger.record.Severity())
	assert.Equal(t, log.StringValue("gorm.sql.error"), provider.logger.record.Body())
	attrs := recordAttributes(provider.logger.record)
	assert.Equal(t, "select * from users", attrs["db.query.text"].Value.AsString())
	assert.Equal(t, int64(3), attrs["db.response.returned_rows"].Value.AsInt64())
	assert.Equal(t, int64(3), attrs["gorm.rows_affected"].Value.AsInt64())
	assert.Equal(t, "gorm.sql.error", attrs["gorm.event"].Value.AsString())
	assert.Equal(t, "*errors.errorString", attrs["error.type"].Value.AsString())
	assert.Equal(t, "database failed", attrs["error.message"].Value.AsString())
}

func TestTraceSlowSQL(t *testing.T) {
	provider := &recordingLoggerProvider{}
	logger := New(
		WithLoggerProvider(provider),
		WithLogLevel(gormlogger.Warn),
		WithSlowThreshold(time.Millisecond),
	)
	provider.logger.enabled = true

	logger.Trace(t.Context(), time.Now().Add(-10*time.Millisecond), func() (string, int64) {
		return "update users set name = ?", -1
	}, nil)

	require.True(t, provider.logger.emitted)
	assert.Equal(t, log.SeverityWarn, provider.logger.record.Severity())
	assert.Equal(t, log.StringValue("gorm.sql.slow"), provider.logger.record.Body())
	attrs := recordAttributes(provider.logger.record)
	assert.Equal(t, "update users set name = ?", attrs["db.query.text"].Value.AsString())
	assert.Equal(t, int64(-1), attrs["gorm.rows_affected"].Value.AsInt64())
	assert.NotContains(t, attrs, "db.response.returned_rows")
}

func TestTraceInfo(t *testing.T) {
	provider := &recordingLoggerProvider{}
	logger := New(WithLoggerProvider(provider), WithLogLevel(gormlogger.Info))
	provider.logger.enabled = true

	logger.Trace(t.Context(), time.Now(), func() (string, int64) {
		return "select 1", 1
	}, nil)

	require.True(t, provider.logger.emitted)
	assert.Equal(t, log.SeverityInfo, provider.logger.record.Severity())
	assert.Equal(t, log.StringValue("gorm.sql"), provider.logger.record.Body())
}

func TestTraceIgnoresRecordNotFound(t *testing.T) {
	provider := &recordingLoggerProvider{}
	logger := New(
		WithLoggerProvider(provider),
		WithLogLevel(gormlogger.Error),
		WithIgnoreRecordNotFoundError(true),
	)
	provider.logger.enabled = true
	called := false

	logger.Trace(t.Context(), time.Now(), func() (string, int64) {
		called = true
		return "select * from users", 0
	}, gormlogger.ErrRecordNotFound)

	assert.False(t, called)
	assert.False(t, provider.logger.emitted)
}

func TestTraceSkipsSQLRenderingWhenDisabled(t *testing.T) {
	provider := &recordingLoggerProvider{}
	logger := New(WithLoggerProvider(provider), WithLogLevel(gormlogger.Info))
	provider.logger.enabled = false
	called := false

	logger.Trace(t.Context(), time.Now(), func() (string, int64) {
		called = true
		return "select 1", 1
	}, nil)

	assert.False(t, called)
	assert.False(t, provider.logger.emitted)
	assert.Equal(t, log.SeverityInfo, provider.logger.enabledParam.Severity)
}

func TestParamsFilter(t *testing.T) {
	logger := New()

	sql, params := logger.ParamsFilter(t.Context(), "select * from users where id = ?", 1)

	assert.Equal(t, "select * from users where id = ?", sql)
	assert.Equal(t, []any{1}, params)
}

func TestParamsFilterParameterizedQueries(t *testing.T) {
	logger := New(WithParameterizedQueries(true))

	sql, params := logger.ParamsFilter(t.Context(), "select * from users where id = ?", 1)

	assert.Equal(t, "select * from users where id = ?", sql)
	assert.Nil(t, params)
}

func recordAttributes(record log.Record) map[string]log.KeyValue {
	attrs := make(map[string]log.KeyValue)
	record.WalkAttributes(func(attr log.KeyValue) bool {
		attrs[attr.Key] = attr
		return true
	})
	return attrs
}
