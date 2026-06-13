package crontab

import (
	"errors"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingLogger struct {
	entries []logEntry
	err     error
}

type logEntry struct {
	level   log.Level
	keyvals []any
}

func (l *recordingLogger) Log(level log.Level, keyvals ...any) error {
	l.entries = append(l.entries, logEntry{
		level:   level,
		keyvals: append([]any(nil), keyvals...),
	})
	return l.err
}

func TestNewLogger_Defaults(t *testing.T) {
	t.Parallel()

	logger := NewLogger()

	require.NotNil(t, logger)
	assert.NotNil(t, logger.logger)
}

func TestNewLogger_Options(t *testing.T) {
	t.Parallel()

	custom := &recordingLogger{}

	tests := []struct {
		name string
		opts []LoggerOption
		want log.Logger
	}{
		{
			name: "uses custom logger",
			opts: []LoggerOption{WithLogger(custom)},
			want: custom,
		},
		{
			name: "ignores nil logger",
			opts: []LoggerOption{WithLogger(nil)},
			want: log.DefaultLogger,
		},
		{
			name: "ignores nil option",
			opts: []LoggerOption{nil},
			want: log.DefaultLogger,
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
	logger := NewLogger(WithLogger(backend))

	logger.Printf("job %s finished in %dms", "sync", 12)

	require.Len(t, backend.entries, 1)
	assert.Equal(t, log.LevelInfo, backend.entries[0].level)
	assert.Equal(t, []any{"msg", "job sync finished in 12ms"}, backend.entries[0].keyvals)
}

func TestLogger_PrintfIgnoresBackendError(t *testing.T) {
	t.Parallel()

	backend := &recordingLogger{err: errors.New("write failed")}
	logger := NewLogger(WithLogger(backend))

	require.NotPanics(t, func() {
		logger.Printf("job failed")
	})
	require.Len(t, backend.entries, 1)
}
