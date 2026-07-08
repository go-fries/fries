package server

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerDoesNotExposeHTTPServer(t *testing.T) {
	serverType := reflect.TypeFor[Server]()
	httpServerType := reflect.TypeFor[*http.Server]()

	for i := range serverType.NumField() {
		field := serverType.Field(i)
		assert.False(t, field.Anonymous && field.Type == httpServerType)
	}
}

func TestNewUsesConfiguredHTTPServer(t *testing.T) {
	srv := New(&http.Server{
		Addr: ":8081",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "server", r.URL.Query().Get("name"))
			_, _ = w.Write([]byte("hello world"))
		}),
	})

	req, err := http.NewRequest(http.MethodGet, "/?name=server", nil)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "hello world", recorder.Body.String())
	assert.Equal(t, ":8081", srv.server.Addr)
}

func TestNewWithHandlerBuildsHTTPServer(t *testing.T) {
	srv := NewWithHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	}), WithAddr(":8082"))

	req, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "hello world", recorder.Body.String())
	assert.Equal(t, ":8082", srv.server.Addr)
}

func TestNewDefaultsToSlogDefaultLogger(t *testing.T) {
	logger := slog.New(&recordingHandler{})
	previous := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(previous)

	srv := New(&http.Server{}, WithLogger(nil))

	assert.Same(t, logger, srv.logger)
}

func TestWithLoggerConfiguresServerLogger(t *testing.T) {
	logger := slog.New(&recordingHandler{})
	srv := New(&http.Server{}, WithLogger(logger))

	assert.Same(t, logger, srv.logger)
}

func TestServerStartWritesToConfiguredLogger(t *testing.T) {
	handler := &recordingHandler{}
	srv := New(&http.Server{Addr: "invalid addr"}, WithLogger(slog.New(handler)))

	err := srv.Start(t.Context())

	require.Error(t, err)
	require.Len(t, handler.records, 1)
	assert.Equal(t, slog.LevelInfo, handler.records[0].Level)
	assert.Equal(t, "[HTTP] server listening on: invalid addr", handler.records[0].Message)
}

func TestServerStopWritesToConfiguredLogger(t *testing.T) {
	handler := &recordingHandler{}
	srv := New(&http.Server{}, WithLogger(slog.New(handler)))

	err := srv.Stop(t.Context())

	require.NoError(t, err)
	require.Len(t, handler.records, 1)
	assert.Equal(t, slog.LevelInfo, handler.records[0].Level)
	assert.Equal(t, "[HTTP] server stopping", handler.records[0].Message)
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
