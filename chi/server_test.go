package chi

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerDoesNotExposeMux(t *testing.T) {
	serverType := reflect.TypeFor[Server]()
	muxType := reflect.TypeFor[*chi.Mux]()

	for i := range serverType.NumField() {
		field := serverType.Field(i)
		assert.False(t, field.Anonymous && field.Type == muxType)
	}
}

func TestServerServesConfiguredRouter(t *testing.T) {
	router := chi.NewRouter()
	router.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte("pong"))
	})
	srv := NewServer(router, WithAddr(":8001"))

	req, err := http.NewRequest(http.MethodGet, "/ping", nil)
	assert.NoError(t, err)

	recorder := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "pong", recorder.Body.String())
	assert.Equal(t, ":8001", srv.server.Addr)
}

func TestNewServerDefaultsToSlogDefaultLogger(t *testing.T) {
	logger := slog.New(&recordingHandler{})
	previous := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(previous)

	srv := NewServer(chi.NewRouter(), WithLogger(nil))

	assert.Same(t, logger, srv.logger)
}

func TestWithLoggerConfiguresServerLogger(t *testing.T) {
	logger := slog.New(&recordingHandler{})
	srv := NewServer(chi.NewRouter(), WithLogger(logger))

	assert.Same(t, logger, srv.logger)
}

func TestServerStartWritesToConfiguredLogger(t *testing.T) {
	handler := &recordingHandler{}
	srv := NewServer(chi.NewRouter(), WithAddr("invalid addr"), WithLogger(slog.New(handler)))

	err := srv.Start(t.Context())

	require.Error(t, err)
	require.Len(t, handler.records, 1)
	assert.Equal(t, slog.LevelInfo, handler.records[0].Level)
	assert.Equal(t, "[go-chi] server listening on: invalid addr", handler.records[0].Message)
}

func TestServerStartReturnsServerClosed(t *testing.T) {
	srv := NewServer(chi.NewRouter(), WithAddr("127.0.0.1:0"))
	require.NoError(t, srv.server.Close())

	err := srv.Start(t.Context())

	require.ErrorIs(t, err, http.ErrServerClosed)
}

func TestServerStopWritesToConfiguredLogger(t *testing.T) {
	handler := &recordingHandler{}
	srv := NewServer(chi.NewRouter(), WithLogger(slog.New(handler)))

	err := srv.Stop(t.Context())

	require.NoError(t, err)
	require.Len(t, handler.records, 1)
	assert.Equal(t, slog.LevelInfo, handler.records[0].Level)
	assert.Equal(t, "[go-chi] server stopping", handler.records[0].Message)
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
