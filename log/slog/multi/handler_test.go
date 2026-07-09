package multi

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerDispatchesEnabledRecordsToMultipleHandlers(t *testing.T) {
	ctx := t.Context()
	first := newRecordingHandler(true)
	second := newRecordingHandler(true)
	disabled := newRecordingHandler(false)
	handler := NewHandler(first, nil, disabled, second)

	record := slog.NewRecord(timeNow(), slog.LevelInfo, "hello", 0)
	record.AddAttrs(slog.String("request_id", "abc123"))

	assert.True(t, handler.Enabled(ctx, slog.LevelInfo))
	require.NoError(t, handler.Handle(ctx, record))

	require.Len(t, *first.records, 1)
	require.Len(t, *second.records, 1)
	assert.Empty(t, *disabled.records)
	assert.Equal(t, "hello", (*first.records)[0].message)
	assert.Equal(t, []slog.Attr{slog.String("request_id", "abc123")}, (*first.records)[0].attrs)
}

func TestHandlerEnabledReturnsFalseWhenNoChildHandlerIsEnabled(t *testing.T) {
	handler := NewHandler(nil, newRecordingHandler(false), newRecordingHandler(false))

	assert.False(t, handler.Enabled(t.Context(), slog.LevelWarn))
}

func TestHandlerJoinsChildHandlerErrors(t *testing.T) {
	firstErr := errors.New("first")
	secondErr := errors.New("second")
	first := newRecordingHandler(true)
	first.err = firstErr
	second := newRecordingHandler(true)
	second.err = secondErr
	handler := NewHandler(first, second)

	err := handler.Handle(t.Context(), slog.NewRecord(timeNow(), slog.LevelError, "failed", 0))

	require.Error(t, err)
	assert.ErrorIs(t, err, firstErr)
	assert.ErrorIs(t, err, secondErr)
}

func TestHandlerWithAttrsReturnsHandlerWithChildAttributes(t *testing.T) {
	child := newRecordingHandler(true)
	handler := NewHandler(child).WithAttrs([]slog.Attr{
		slog.String("service", "api"),
		slog.Int("attempt", 2),
	})

	require.NoError(t, handler.Handle(t.Context(), slog.NewRecord(timeNow(), slog.LevelInfo, "hello", 0)))

	require.Len(t, *child.records, 1)
	assert.Equal(t, []slog.Attr{slog.String("service", "api"), slog.Int("attempt", 2)}, (*child.records)[0].handlerAttrs)
}

func TestHandlerWithAttrsReturnsReceiverWhenAttrsEmpty(t *testing.T) {
	handler := NewHandler(newRecordingHandler(true))

	assert.Same(t, handler, handler.WithAttrs(nil))
	assert.Same(t, handler, handler.WithAttrs([]slog.Attr{}))
}

func TestHandlerWithAttrsIgnoresNilDerivedChildHandlers(t *testing.T) {
	handler := NewHandler(&nilDerivedHandler{}).WithAttrs([]slog.Attr{slog.String("service", "api")})

	require.NotPanics(t, func() {
		assert.False(t, handler.Enabled(t.Context(), slog.LevelInfo))
	})
	require.NotPanics(t, func() {
		assert.NoError(t, handler.Handle(t.Context(), slog.NewRecord(timeNow(), slog.LevelInfo, "hello", 0)))
	})
}

func TestHandlerWithGroupReturnsHandlerWithChildGroup(t *testing.T) {
	child := newRecordingHandler(true)
	handler := NewHandler(child).WithGroup("queue")

	require.NoError(t, handler.Handle(t.Context(), slog.NewRecord(timeNow(), slog.LevelInfo, "hello", 0)))

	require.Len(t, *child.records, 1)
	assert.Equal(t, []string{"queue"}, (*child.records)[0].groups)
}

func TestHandlerWithGroupReturnsReceiverWhenNameEmpty(t *testing.T) {
	handler := NewHandler(newRecordingHandler(true))

	assert.Same(t, handler, handler.WithGroup(""))
}

func TestHandlerWithGroupIgnoresNilDerivedChildHandlers(t *testing.T) {
	handler := NewHandler(&nilDerivedHandler{}).WithGroup("queue")

	require.NotPanics(t, func() {
		assert.False(t, handler.Enabled(t.Context(), slog.LevelInfo))
	})
	require.NotPanics(t, func() {
		assert.NoError(t, handler.Handle(t.Context(), slog.NewRecord(timeNow(), slog.LevelInfo, "hello", 0)))
	})
}

func TestHandlerWithAttrsAndWithGroupDoNotMutateOriginalHandler(t *testing.T) {
	child := newRecordingHandler(true)
	handler := NewHandler(child)
	derived := handler.WithAttrs([]slog.Attr{slog.String("component", "worker")}).WithGroup("job")

	require.NoError(t, handler.Handle(t.Context(), slog.NewRecord(timeNow(), slog.LevelInfo, "base", 0)))
	require.NoError(t, derived.Handle(t.Context(), slog.NewRecord(timeNow(), slog.LevelInfo, "derived", 0)))

	require.Len(t, *child.records, 2)
	assert.Empty(t, (*child.records)[0].handlerAttrs)
	assert.Empty(t, (*child.records)[0].groups)
	assert.Equal(t, []slog.Attr{slog.String("component", "worker")}, (*child.records)[1].handlerAttrs)
	assert.Equal(t, []string{"job"}, (*child.records)[1].groups)
}

type recordedRecord struct {
	message      string
	attrs        []slog.Attr
	handlerAttrs []slog.Attr
	groups       []string
}

type recordingHandler struct {
	enabled bool
	err     error
	attrs   []slog.Attr
	groups  []string
	records *[]recordedRecord
}

func newRecordingHandler(enabled bool) *recordingHandler {
	records := make([]recordedRecord, 0)
	return &recordingHandler{enabled: enabled, records: &records}
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool {
	return h.enabled
}

func (h *recordingHandler) Handle(_ context.Context, record slog.Record) error {
	attrs := make([]slog.Attr, 0, record.NumAttrs())
	record.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})
	*h.records = append(*h.records, recordedRecord{
		message:      record.Message,
		attrs:        slices.Clone(attrs),
		handlerAttrs: slices.Clone(h.attrs),
		groups:       slices.Clone(h.groups),
	})
	return h.err
}

func (h *recordingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = append(slices.Clone(h.attrs), attrs...)
	return &clone
}

func (h *recordingHandler) WithGroup(name string) slog.Handler {
	clone := *h
	clone.groups = append(slices.Clone(h.groups), name)
	return &clone
}

type nilDerivedHandler struct{}

func (h *nilDerivedHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *nilDerivedHandler) Handle(context.Context, slog.Record) error {
	return nil
}

func (h *nilDerivedHandler) WithAttrs([]slog.Attr) slog.Handler {
	return nil
}

func (h *nilDerivedHandler) WithGroup(string) slog.Handler {
	return nil
}

func timeNow() time.Time {
	return time.Unix(1710000000, 0)
}
