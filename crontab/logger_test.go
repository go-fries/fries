package crontab

import (
	"fmt"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testLogger struct {
	ch chan string
}

func newTestLogger() *testLogger {
	return &testLogger{
		ch: make(chan string, 1),
	}
}

func (t *testLogger) Log(level log.Level, keyvals ...any) error {
	t.ch <- fmt.Sprintf("level: %s, keyvals: %v", level, keyvals)
	return nil
}

func TestLogger(t *testing.T) {
	_, ok := any(NewLogger()).(interface{ Printf(string, ...any) })
	assert.True(t, ok)

	logger := newTestLogger()
	l := NewLogger(WithLogger(logger))
	l.Printf("test %s", "logger")
	assert.Equal(t, "level: INFO, keyvals: [msg test logger]", <-logger.ch)
}

func TestLogger_UsesDefaultLogger(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		NewLogger().Printf("test %s", "logger")
	})
}

func TestLogger_WithLoggerIgnoresNilLogger(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		NewLogger(WithLogger(nil)).Printf("test %s", "logger")
	})
}

func TestLogger_IgnoresNilOption(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		NewLogger(nil).Printf("test %s", "logger")
	})
}

func TestLogger_ZeroValueDoesNotPanic(t *testing.T) {
	t.Parallel()

	var logger Logger

	require.NotPanics(t, func() {
		logger.Printf("test %s", "logger")
	})
}
