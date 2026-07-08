package logger

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingHandler struct {
	records []slog.Record
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *recordingHandler) Handle(_ context.Context, record slog.Record) error {
	h.records = append(h.records, record.Clone())
	return nil
}

func (h *recordingHandler) WithAttrs([]slog.Attr) slog.Handler {
	return h
}

func (h *recordingHandler) WithGroup(string) slog.Handler {
	return h
}

func TestLogging(t *testing.T) {
	handler := &recordingHandler{}
	middleware := New(
		WithLogger(slog.New(handler)),
	)

	_, _ = middleware(func(_ context.Context, service, method string, _ any) (response any, err error) {
		assert.Equal(t, "service", service)
		assert.Equal(t, "no-error", method)
		return "response", nil
	})(t.Context(), "service", "no-error", nil)

	// with error
	_, _ = middleware(func(_ context.Context, service, method string, _ any) (response any, err error) {
		assert.Equal(t, "service", service)
		assert.Equal(t, "with-error", method)
		return nil, assert.AnError
	})(t.Context(), "service", "with-error", nil)

	require.Len(t, handler.records, 2)
	assert.Equal(t, slog.LevelInfo, handler.records[0].Level)
	assert.Equal(t, slog.LevelError, handler.records[1].Level)

	noErrorAttrs := attrsMap(handler.records[0])
	assert.Equal(t, "jet", noErrorAttrs["kind"])
	assert.Equal(t, "service", noErrorAttrs["service"])
	assert.Equal(t, "no-error", noErrorAttrs["method"])
	assert.Nil(t, noErrorAttrs["request"])
	assert.Equal(t, "response", noErrorAttrs["response"])
	assert.Nil(t, noErrorAttrs["error"])
	assert.IsType(t, time.Duration(0), noErrorAttrs["latency"])

	withErrorAttrs := attrsMap(handler.records[1])
	assert.Equal(t, "with-error", withErrorAttrs["method"])
	assert.Nil(t, withErrorAttrs["response"])
	assert.Equal(t, assert.AnError, withErrorAttrs["error"])
}

func TestNewSkipsNilOptions(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		middleware := New(nil, WithLogger(nil))
		_, _ = middleware(func(context.Context, string, string, any) (any, error) {
			return nil, nil
		})(t.Context(), "service", "method", nil)
	})
}

func attrsMap(record slog.Record) map[string]any {
	attrs := make(map[string]any, record.NumAttrs())
	record.Attrs(func(attr slog.Attr) bool {
		attrs[attr.Key] = attr.Value.Any()
		return true
	})
	return attrs
}
