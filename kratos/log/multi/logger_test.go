package multi

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingLogger struct {
	logs []string
	err  error
}

var _ log.Logger = (*recordingLogger)(nil)

func (r *recordingLogger) Log(level log.Level, keyvals ...any) error {
	r.logs = append(r.logs, fmt.Sprintf("level: %v, keyvals: %v", level, keyvals))
	return r.err
}

func TestNewSkipsNilLoggers(t *testing.T) {
	logger := New(nil)

	assert.Empty(t, logger.loggers)
}

func TestLogNoLoggers(t *testing.T) {
	logger := New()

	err := logger.Log(log.LevelInfo, "key", "value")

	assert.NoError(t, err)
}

func TestLogDispatchesToAllLoggers(t *testing.T) {
	first := &recordingLogger{}
	second := &recordingLogger{}
	logger := New(first, second)

	err := logger.Log(log.LevelInfo, "key", "value")

	require.NoError(t, err)
	expected := "level: INFO, keyvals: [key value]"
	assert.Equal(t, []string{expected}, first.logs)
	assert.Equal(t, []string{expected}, second.logs)
}

func TestLogReturnsSingleLoggerError(t *testing.T) {
	want := errors.New("write failed")
	item := &recordingLogger{err: want}
	logger := New(item)

	err := logger.Log(log.LevelError, "key", "value")

	assert.ErrorIs(t, err, want)
	assert.Len(t, item.logs, 1)
}

func TestLogJoinsMultipleErrorsAndContinuesDispatch(t *testing.T) {
	firstErr := errors.New("first failed")
	secondErr := errors.New("second failed")
	first := &recordingLogger{err: firstErr}
	second := &recordingLogger{err: secondErr}
	third := &recordingLogger{}
	logger := New(first, second, third)

	err := logger.Log(log.LevelWarn, "key", "value")

	require.Error(t, err)
	assert.ErrorIs(t, err, firstErr)
	assert.ErrorIs(t, err, secondErr)
	assert.Len(t, first.logs, 1)
	assert.Len(t, second.logs, 1)
	assert.Len(t, third.logs, 1)
}
