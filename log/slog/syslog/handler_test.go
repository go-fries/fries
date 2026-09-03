//go:build !windows && !plan9

package syslog

import (
	"bytes"
	"errors"
	"log/slog"
	stdsyslog "log/syslog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerWritesRecordToInjectedWriter(t *testing.T) {
	writer := &recordingWriter{}
	handler := NewHandler(writer)
	record := slog.NewRecord(timeNow(), slog.LevelInfo, "service started", 0)
	record.AddAttrs(slog.String("service", "api"), slog.Int("attempt", 2))

	require.NoError(t, handler.Handle(t.Context(), record))

	require.Len(t, writer.records, 1)
	assert.Equal(t, "info", writer.records[0].priority)
	assert.Contains(t, writer.records[0].message, "service started")
	assert.Contains(t, writer.records[0].message, "service=api")
	assert.Contains(t, writer.records[0].message, "attempt=2")
}

func TestHandlerMapsSlogLevelsToSyslogPriorities(t *testing.T) {
	tests := []struct {
		name     string
		level    slog.Level
		priority string
	}{
		{name: "debug", level: slog.LevelDebug, priority: "debug"},
		{name: "below info", level: slog.LevelInfo - 1, priority: "debug"},
		{name: "info", level: slog.LevelInfo, priority: "info"},
		{name: "warn", level: slog.LevelWarn, priority: "warning"},
		{name: "error", level: slog.LevelError, priority: "err"},
		{name: "above error", level: slog.LevelError + 4, priority: "err"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := &recordingWriter{}
			handler := NewHandler(writer)

			require.NoError(t, handler.Handle(t.Context(), slog.NewRecord(timeNow(), tt.level, tt.name, 0)))

			require.Len(t, writer.records, 1)
			assert.Equal(t, tt.priority, writer.records[0].priority)
		})
	}
}

func TestHandlerWithAttrsAndWithGroupQualifiesAttributes(t *testing.T) {
	writer := &recordingWriter{}
	handler := NewHandler(writer).
		WithAttrs([]slog.Attr{slog.String("service", "api")}).
		WithGroup("worker")
	record := slog.NewRecord(timeNow(), slog.LevelWarn, "job failed", 0)
	record.AddAttrs(slog.Int("attempt", 3))

	require.NoError(t, handler.Handle(t.Context(), record))

	require.Len(t, writer.records, 1)
	assert.Equal(t, "warning", writer.records[0].priority)
	assert.Contains(t, writer.records[0].message, "service=api")
	assert.Contains(t, writer.records[0].message, "worker.attempt=3")
}

func TestHandlerWritesGroupAttributes(t *testing.T) {
	writer := &recordingWriter{}
	handler := NewHandler(writer)
	record := slog.NewRecord(timeNow(), slog.LevelInfo, "job finished", 0)
	record.AddAttrs(slog.Group(
		"worker",
		slog.String("name", "syncer"),
		slog.Group("job", slog.Int("attempt", 3)),
		slog.Group("empty"),
	))

	require.NoError(t, handler.Handle(t.Context(), record))

	require.Len(t, writer.records, 1)
	assert.Contains(t, writer.records[0].message, "worker.name=syncer")
	assert.Contains(t, writer.records[0].message, "worker.job.attempt=3")
	assert.NotContains(t, writer.records[0].message, "empty")
}

func TestHandlerQuotesStringAttributesWhenNeeded(t *testing.T) {
	writer := &recordingWriter{}
	handler := NewHandler(writer)
	record := slog.NewRecord(timeNow(), slog.LevelInfo, "service started", 0)
	record.AddAttrs(
		slog.String("plain", "api"),
		slog.String("empty", ""),
		slog.String("space", "api worker"),
		slog.String("equals", "a=b"),
		slog.String("quote", `say "hello"`),
	)

	require.NoError(t, handler.Handle(t.Context(), record))

	require.Len(t, writer.records, 1)
	assert.Contains(t, writer.records[0].message, "plain=api")
	assert.Contains(t, writer.records[0].message, `empty=""`)
	assert.Contains(t, writer.records[0].message, `space="api worker"`)
	assert.Contains(t, writer.records[0].message, `equals="a=b"`)
	assert.Contains(t, writer.records[0].message, `quote="say \"hello\""`)
}

func TestHandlerSerializesConcurrentWrites(t *testing.T) {
	writer := &serialWriter{}
	handler := NewHandler(writer)
	derived := handler.WithGroup("worker")

	var wg sync.WaitGroup
	for range 64 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			assert.NoError(t, handler.Handle(t.Context(), slog.NewRecord(timeNow(), slog.LevelInfo, "base", 0)))
		}()
		go func() {
			defer wg.Done()
			assert.NoError(t, derived.Handle(t.Context(), slog.NewRecord(timeNow(), slog.LevelInfo, "derived", 0)))
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(128), writer.count.Load())
}

func TestWithFacilityMasksSeverityBits(t *testing.T) {
	cfg := newConfig(WithFacility(stdsyslog.LOG_LOCAL0 | stdsyslog.LOG_INFO))

	assert.Equal(t, stdsyslog.LOG_LOCAL0, cfg.facility)
}

func TestConfigOptions(t *testing.T) {
	level := new(slog.LevelVar)
	level.Set(slog.LevelWarn)
	cfg := newConfig(nil, WithLevel(level), WithTag("fries"))

	assert.Equal(t, stdsyslog.LOG_USER, cfg.facility)
	assert.Same(t, level, cfg.level)
	assert.Equal(t, "fries", cfg.tag)
}

func TestHandlerEnabledHonorsMinimumLevel(t *testing.T) {
	writer := &recordingWriter{}
	handler := NewHandler(writer, WithLevel(slog.LevelWarn))

	assert.False(t, handler.Enabled(t.Context(), slog.LevelInfo))
	assert.True(t, handler.Enabled(t.Context(), slog.LevelError))
	require.NoError(t, handler.Handle(t.Context(), slog.NewRecord(timeNow(), slog.LevelInfo, "ignore", 0)))
	assert.Empty(t, writer.records)
}

func TestHandlerReturnsWriterError(t *testing.T) {
	writerErr := errors.New("write failed")
	writer := &recordingWriter{err: writerErr}
	handler := NewHandler(writer)

	err := handler.Handle(t.Context(), slog.NewRecord(timeNow(), slog.LevelError, "failed", 0))

	assert.ErrorIs(t, err, writerErr)
}

func TestHandlerWithEmptyAttrsAndGroupReturnsReceiver(t *testing.T) {
	handler := NewHandler(&recordingWriter{})

	assert.Same(t, handler, handler.WithAttrs(nil))
	assert.Same(t, handler, handler.WithAttrs([]slog.Attr{}))
	assert.Same(t, handler, handler.WithGroup(""))
}

func TestHandlerNoopsWhenWriterNil(t *testing.T) {
	handler := NewHandler(nil)

	assert.False(t, handler.Enabled(t.Context(), slog.LevelInfo))
	assert.NoError(t, handler.Handle(t.Context(), slog.NewRecord(timeNow(), slog.LevelInfo, "ignored", 0)))
}

func TestHandlerClose(t *testing.T) {
	t.Run("writer without close", func(t *testing.T) {
		handler := NewHandler(&recordingWriter{})

		require.NoError(t, handler.Close())
	})

	t.Run("writer close success", func(t *testing.T) {
		writer := &closingWriter{recordingWriter: &recordingWriter{}}
		handler := NewHandler(writer)

		require.NoError(t, handler.Close())
		assert.Equal(t, 1, writer.closeCalls)
	})

	t.Run("writer close error", func(t *testing.T) {
		sentinel := errors.New("close failed")
		writer := &closingWriter{
			recordingWriter: &recordingWriter{},
			closeErr:        sentinel,
		}
		handler := NewHandler(writer)

		err := handler.Close()

		assert.ErrorIs(t, err, sentinel)
		assert.Equal(t, 1, writer.closeCalls)
	})
}

func TestHandlerFormatsValueKinds(t *testing.T) {
	writer := &recordingWriter{}
	handler := NewHandler(writer)
	record := slog.NewRecord(timeNow(), slog.LevelInfo, "values", 0)
	record.AddAttrs(
		slog.Uint64("uint", 42),
		slog.Float64("float", 1.5),
		slog.Bool("bool", true),
		slog.Duration("duration", 2*time.Second),
		slog.Time("time", timeNow()),
		slog.Any("any", []int{1, 2}),
	)

	require.NoError(t, handler.Handle(t.Context(), record))

	require.Len(t, writer.records, 1)
	assert.Contains(t, writer.records[0].message, "uint=42")
	assert.Contains(t, writer.records[0].message, "float=1.5")
	assert.Contains(t, writer.records[0].message, "bool=true")
	assert.Contains(t, writer.records[0].message, "duration=2s")
	assert.Contains(t, writer.records[0].message, "time="+timeNow().Format(time.RFC3339))
	assert.Contains(t, writer.records[0].message, "any=[1 2]")
}

func TestAppendAttrIgnoresEmptyAttributes(t *testing.T) {
	var buf bytes.Buffer

	appendAttr(&buf, nil, slog.Attr{})
	appendAttr(&buf, nil, slog.String("", "value"))
	appendAttr(&buf, nil, slog.Group("empty"))

	assert.Empty(t, buf.String())
}

type recordedWrite struct {
	priority string
	message  string
}

type recordingWriter struct {
	records []recordedWrite
	err     error
}

type closingWriter struct {
	*recordingWriter
	closeErr   error
	closeCalls int
}

func (w *closingWriter) Close() error {
	w.closeCalls++
	return w.closeErr
}

func (w *recordingWriter) Debug(message string) error {
	return w.record("debug", message)
}

func (w *recordingWriter) Info(message string) error {
	return w.record("info", message)
}

func (w *recordingWriter) Warning(message string) error {
	return w.record("warning", message)
}

func (w *recordingWriter) Err(message string) error {
	return w.record("err", message)
}

func (w *recordingWriter) record(priority, message string) error {
	w.records = append(w.records, recordedWrite{
		priority: priority,
		message:  message,
	})
	return w.err
}

func timeNow() time.Time {
	return time.Unix(1710000000, 0)
}

type serialWriter struct {
	active atomic.Int32
	count  atomic.Int32
}

func (w *serialWriter) Debug(string) error {
	return w.record()
}

func (w *serialWriter) Info(string) error {
	return w.record()
}

func (w *serialWriter) Warning(string) error {
	return w.record()
}

func (w *serialWriter) Err(string) error {
	return w.record()
}

func (w *serialWriter) record() error {
	if w.active.Add(1) != 1 {
		w.active.Add(-1)
		return errors.New("concurrent write")
	}
	defer w.active.Add(-1)

	time.Sleep(time.Millisecond)
	w.count.Add(1)
	return nil
}
