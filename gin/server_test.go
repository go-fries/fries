package gin

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func middleware1(t *testing.T) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("middleware1", "middleware1")
		assert.Equal(t, "/ping", c.Request.URL.Path)
		c.Next()
	}
}

func middleware2(t *testing.T) gin.HandlerFunc {
	return func(c *gin.Context) {
		assert.Equal(t, "middleware1", c.MustGet("middleware1").(string))
		c.Set("middleware2", "middleware2")
		assert.Equal(t, "/ping", c.Request.URL.Path)
		c.Next()
	}
}

func TestServerDoesNotExposeEngine(t *testing.T) {
	serverType := reflect.TypeFor[Server]()
	engineType := reflect.TypeFor[*gin.Engine]()

	for i := range serverType.NumField() {
		field := serverType.Field(i)
		assert.False(t, field.Anonymous && field.Type == engineType)
	}
}

func TestServerServesConfiguredEngine(t *testing.T) {
	engine := gin.New()
	engine.Use(middleware1(t), middleware2(t))
	engine.GET("/ping", func(c *gin.Context) {
		assert.Equal(t, http.MethodGet, c.Request.Method)
		assert.Equal(t, "/ping", c.Request.URL.Path)
		assert.Equal(t, "middleware1", c.MustGet("middleware1").(string))
		assert.Equal(t, "middleware2", c.MustGet("middleware2").(string))
		c.String(http.StatusOK, "pong")
	})

	srv := NewServer(engine, WithAddr(":8080"))

	req, err := http.NewRequest(http.MethodGet, "/ping", nil)
	assert.NoError(t, err)

	recorder := httptest.NewRecorder()
	srv.server.Handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "pong", recorder.Body.String())
	assert.Equal(t, ":8080", srv.server.Addr)
}

func TestNewServerDefaultsToSlogDefaultLogger(t *testing.T) {
	logger := slog.New(&recordingHandler{})
	previous := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(previous)

	srv := NewServer(gin.New(), WithLogger(nil))

	assert.Same(t, logger, srv.logger)
}

func TestWithLoggerConfiguresServerLogger(t *testing.T) {
	logger := slog.New(&recordingHandler{})
	srv := NewServer(gin.New(), WithLogger(logger))

	assert.Same(t, logger, srv.logger)
}

func TestServerStartWritesToConfiguredLogger(t *testing.T) {
	handler := &recordingHandler{}
	srv := NewServer(gin.New(), WithAddr("invalid addr"), WithLogger(slog.New(handler)))

	err := srv.Start(t.Context())

	require.Error(t, err)
	require.Len(t, handler.records, 1)
	assert.Equal(t, slog.LevelInfo, handler.records[0].Level)
	assert.Equal(t, "[GIN] server listening on: invalid addr", handler.records[0].Message)
}

func TestServerStopWritesToConfiguredLogger(t *testing.T) {
	handler := &recordingHandler{}
	srv := NewServer(gin.New(), WithLogger(slog.New(handler)))

	err := srv.Stop(t.Context())

	require.NoError(t, err)
	require.Len(t, handler.records, 1)
	assert.Equal(t, slog.LevelInfo, handler.records[0].Level)
	assert.Equal(t, "[GIN] server stopping", handler.records[0].Message)
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
