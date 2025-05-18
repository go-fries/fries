package stack

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
)

type mockLogger struct {
	logs []string
	err  error
}

func (m *mockLogger) Log(level log.Level, keyvals ...any) error {
	if m.err != nil {
		return m.err
	}
	m.logs = append(m.logs, formatLog(level, keyvals...))
	return nil
}

func formatLog(level log.Level, keyvals ...any) string {
	return fmt.Sprintf("level: %v, keyvals: %v", level, keyvals)
}

func TestStackLogger_Log(t *testing.T) {
	logger1 := &mockLogger{}
	logger2 := &mockLogger{}
	stack := New(logger1, logger2)

	err := stack.Log(log.LevelInfo, "key", "value")
	assert.NoError(t, err)
	assert.Len(t, logger1.logs, 1)
	assert.Len(t, logger2.logs, 1)

	expectedLog := "level: INFO, keyvals: [key value]"
	assert.Equal(t, expectedLog, logger1.logs[0])
	assert.Equal(t, expectedLog, logger2.logs[0])
}

func TestStackLogger_LogWithError(t *testing.T) {
	logger1 := &mockLogger{}
	logger2 := &mockLogger{err: errors.New("mock error")}
	stack := New(logger1, logger2)

	err := stack.Log(log.LevelInfo, "key", "value")
	assert.Error(t, err)
	assert.Equal(t, "mock error", err.Error())

	assert.Len(t, logger1.logs, 1)
	assert.Len(t, logger2.logs, 0)

	expectedLog := "level: INFO, keyvals: [key value]"
	assert.Equal(t, expectedLog, logger1.logs[0])
}

func TestStackLogger_LogWithMultipleErrors(t *testing.T) {
	logger1 := &mockLogger{err: errors.New("error 1")}
	logger2 := &mockLogger{err: errors.New("error 2")}
	stack := New(logger1, logger2)

	err := stack.Log(log.LevelInfo, "key", "value")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error 1")
	assert.Contains(t, err.Error(), "error 2")

	assert.Len(t, logger1.logs, 0)
	assert.Len(t, logger2.logs, 0)
}
