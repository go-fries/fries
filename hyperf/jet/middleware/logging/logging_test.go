package logging

import (
	"context"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
)

type mockLogger struct {
	t *testing.T
}

func newMockLogger(t *testing.T) log.Logger {
	return &mockLogger{t: t}
}

func (m *mockLogger) Log(level log.Level, keyvals ...any) error {
	if !assert.Len(m.t, keyvals, 14) {
		return nil
	}

	assert.Equal(m.t, "kind", keyvals[0])
	assert.Equal(m.t, "jet", keyvals[1])
	assert.Equal(m.t, "service", keyvals[2])
	assert.Equal(m.t, "service", keyvals[3])
	assert.Equal(m.t, "method", keyvals[4])

	switch keyvals[5] {
	case "no-error":
		assert.Equal(m.t, log.LevelInfo, level)
		assert.Equal(m.t, "response", keyvals[9])
		assert.Nil(m.t, keyvals[11])
	case "with-error":
		assert.Equal(m.t, log.LevelError, level)
		assert.Nil(m.t, keyvals[9])
		assert.Equal(m.t, assert.AnError, keyvals[11])
	default:
		assert.Fail(m.t, "unexpected name")
	}

	assert.Equal(m.t, "request", keyvals[6])
	assert.Nil(m.t, keyvals[7])
	assert.Equal(m.t, "response", keyvals[8])
	assert.Equal(m.t, "error", keyvals[10])
	assert.Equal(m.t, "latency", keyvals[12])
	assert.IsType(m.t, time.Duration(0), keyvals[13])
	return nil
}

var _ log.Logger = (*mockLogger)(nil)

func TestLogging(t *testing.T) {
	logging := New(
		Logger(newMockLogger(t)),
	)

	// no error
	_, _ = logging(func(_ context.Context, service, method string, _ any) (response any, err error) {
		assert.Equal(t, "service", service)
		assert.Equal(t, "no-error", method)
		return "response", nil
	})(t.Context(), "service", "no-error", nil)

	// with error
	_, _ = logging(func(_ context.Context, service, method string, _ any) (response any, err error) {
		assert.Equal(t, "service", service)
		assert.Equal(t, "with-error", method)
		return nil, assert.AnError
	})(t.Context(), "service", "with-error", nil)
}
