package ent

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoggerWritesToSlogLogger(t *testing.T) {
	handler := &recordingHandler{}
	logger := NewLogger(slog.New(handler), slog.LevelInfo)

	logger("test", " logger")

	require.Len(t, handler.records, 1)
	assert.Equal(t, slog.LevelInfo, handler.records[0].Level)
	assert.Equal(t, "test logger", handler.records[0].Message)
}

func TestNewLoggerDefaultsToSlogDefault(t *testing.T) {
	handler := &recordingHandler{}
	previous := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(previous)

	logger := NewLogger(nil)
	logger("default")

	require.Len(t, handler.records, 1)
	assert.Equal(t, slog.LevelDebug, handler.records[0].Level)
	assert.Equal(t, "default", handler.records[0].Message)
}

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
