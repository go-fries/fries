package server

import (
	"context"
	"log/slog"
	"testing"

	"github.com/go-fries/fries/mysql/canal/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_StartWritesToConfiguredLogger(t *testing.T) {
	t.Parallel()

	handler := &recordingHandler{}
	cnl, err := canal.New(&canal.Config{})
	require.NoError(t, err)
	server := New(cnl, WithLogger(slog.New(handler)))

	err = server.Start(t.Context())

	require.Error(t, err)
	require.Len(t, handler.records, 1)
	assert.Equal(t, slog.LevelInfo, handler.records[0].Level)
	assert.Equal(t, "[Canal] server starting", handler.records[0].Message)
}

func TestServer_StopWritesToConfiguredLogger(t *testing.T) {
	t.Parallel()

	handler := &recordingHandler{}
	cnl, err := canal.New(&canal.Config{})
	require.NoError(t, err)
	server := New(cnl, WithLogger(slog.New(handler)))

	require.NoError(t, server.Stop(t.Context()))
	require.Len(t, handler.records, 1)
	assert.Equal(t, slog.LevelInfo, handler.records[0].Level)
	assert.Equal(t, "[Canal] server stopping", handler.records[0].Message)
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
