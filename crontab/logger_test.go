package crontab

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingLogger struct {
	records []slog.Record
	err     error
}

type loggerOnlyOption struct{}

func (loggerOnlyOption) applyLogger(*loggerConfig) {}

func requireLoggerServerOption(option LoggerServerOption) LoggerServerOption {
	return option
}

func (l *recordingLogger) Enabled(context.Context, slog.Level) bool {
	return true
}

func (l *recordingLogger) Handle(_ context.Context, record slog.Record) error {
	l.records = append(l.records, record.Clone())
	return l.err
}

func (l *recordingLogger) WithAttrs([]slog.Attr) slog.Handler {
	return l
}

func (l *recordingLogger) WithGroup(string) slog.Handler {
	return l
}

func TestNewLogger_Defaults(t *testing.T) {
	t.Parallel()

	logger := NewLogger()

	require.NotNil(t, logger)
	assert.NotNil(t, logger.logger)
}

func TestNewLogger_AcceptsLoggerOnlyOption(t *testing.T) {
	t.Parallel()

	logger := NewLogger(loggerOnlyOption{})

	require.NotNil(t, logger)
	assert.NotNil(t, logger.logger)
}

func TestWithLoggerReturnsLoggerServerOption(t *testing.T) {
	t.Parallel()

	option := requireLoggerServerOption(WithLogger(slog.New(&recordingLogger{})))

	assert.NotNil(t, option)
}

func TestNewLogger_Options(t *testing.T) {
	t.Parallel()

	custom := slog.New(&recordingLogger{})

	tests := []struct {
		name string
		opts []LoggerOption
		want *slog.Logger
	}{
		{
			name: "uses custom logger",
			opts: []LoggerOption{WithLogger(custom)},
			want: custom,
		},
		{
			name: "ignores nil logger",
			opts: []LoggerOption{WithLogger(nil)},
			want: slog.Default(),
		},
		{
			name: "ignores nil option",
			opts: []LoggerOption{nil},
			want: slog.Default(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := NewLogger(tt.opts...)

			assert.Same(t, tt.want, logger.logger)
		})
	}
}

func TestLogger_Printf(t *testing.T) {
	t.Parallel()

	backend := &recordingLogger{}
	logger := NewLogger(WithLogger(slog.New(backend)))

	logger.Printf("job %s finished in %dms", "sync", 12)

	require.Len(t, backend.records, 1)
	assert.Equal(t, slog.LevelInfo, backend.records[0].Level)
	assert.Equal(t, "job sync finished in 12ms", backend.records[0].Message)
}

func TestLogger_PrintfIgnoresBackendError(t *testing.T) {
	t.Parallel()

	backend := &recordingLogger{err: errors.New("write failed")}
	logger := NewLogger(WithLogger(slog.New(backend)))

	require.NotPanics(t, func() {
		logger.Printf("job failed")
	})
	require.Len(t, backend.records, 1)
}
