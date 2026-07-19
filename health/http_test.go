package health_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/go-fries/fries/health/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type healthResponse struct {
	Status string `json:"status"`
	Checks []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
		Error  string `json:"error"`
	} `json:"checks"`
}

func TestHandlerHealthy(t *testing.T) {
	registry := health.New()
	registry.Register("database", health.CheckFunc(func(context.Context) error {
		return nil
	}))

	recorder := httptest.NewRecorder()
	health.Handler(registry).ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/readyz", nil),
	)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

	var response healthResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, "healthy", response.Status)
	require.Len(t, response.Checks, 1)
	assert.Equal(t, "database", response.Checks[0].Name)
	assert.Equal(t, "healthy", response.Checks[0].Status)
	assert.Empty(t, response.Checks[0].Error)
}

func TestHandlerUnhealthyHidesErrorsByDefault(t *testing.T) {
	registry := health.New()
	registry.Register("database", health.CheckFunc(func(context.Context) error {
		return assert.AnError
	}))

	recorder := httptest.NewRecorder()
	health.Handler(registry).ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/readyz", nil),
	)

	assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), assert.AnError.Error())

	var response healthResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, "unhealthy", response.Status)
	require.Len(t, response.Checks, 1)
	assert.Equal(t, "unhealthy", response.Checks[0].Status)
	assert.Empty(t, response.Checks[0].Error)
}

func TestHandlerWithErrorDetails(t *testing.T) {
	registry := health.New()
	registry.Register("database", health.CheckFunc(func(context.Context) error {
		return assert.AnError
	}))

	recorder := httptest.NewRecorder()
	health.Handler(registry, nil, health.WithErrorDetails()).ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/readyz", nil),
	)

	var response healthResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response.Checks, 1)
	assert.Equal(t, assert.AnError.Error(), response.Checks[0].Error)
}

func TestHandlerHead(t *testing.T) {
	registry := health.New()

	getRecorder := httptest.NewRecorder()
	health.Handler(registry).ServeHTTP(
		getRecorder,
		httptest.NewRequest(http.MethodGet, "/livez", nil),
	)

	headRecorder := httptest.NewRecorder()
	health.Handler(registry).ServeHTTP(
		headRecorder,
		httptest.NewRequest(http.MethodHead, "/livez", nil),
	)

	getContentLength, err := strconv.Atoi(getRecorder.Header().Get("Content-Length"))
	require.NoError(t, err)
	assert.Equal(t, getRecorder.Body.Len(), getContentLength)

	assert.Equal(t, http.StatusOK, headRecorder.Code)
	assert.Equal(t, "application/json", headRecorder.Header().Get("Content-Type"))
	headContentLength, err := strconv.Atoi(headRecorder.Header().Get("Content-Length"))
	require.NoError(t, err)
	assert.Positive(t, headContentLength)
	assert.Empty(t, headRecorder.Body.String())
}

func TestHandlerRejectsUnsupportedMethods(t *testing.T) {
	registry := health.New()
	var calls atomic.Int32
	registry.Register("database", health.CheckFunc(func(context.Context) error {
		calls.Add(1)
		return nil
	}))

	recorder := httptest.NewRecorder()
	health.Handler(registry).ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodPost, "/readyz", nil),
	)

	assert.Equal(t, http.StatusMethodNotAllowed, recorder.Code)
	assert.Equal(t, "GET, HEAD", recorder.Header().Get("Allow"))
	assert.Equal(t, int32(0), calls.Load())
}

func TestHandlerPassesRequestContext(t *testing.T) {
	type contextKey struct{}

	registry := health.New()
	registry.Register("context", health.CheckFunc(func(ctx context.Context) error {
		assert.Equal(t, "value", ctx.Value(contextKey{}))
		return nil
	}))

	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	request = request.WithContext(context.WithValue(request.Context(), contextKey{}, "value"))
	recorder := httptest.NewRecorder()

	health.Handler(registry).ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestHandlerPreservesRegistrationOrder(t *testing.T) {
	registry := health.New(health.WithConcurrency(3))
	for _, name := range []string{"first", "second", "third"} {
		registry.Register(name, health.CheckFunc(func(context.Context) error {
			return nil
		}))
	}

	recorder := httptest.NewRecorder()
	health.Handler(registry).ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/readyz", nil),
	)

	var response healthResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response.Checks, 3)
	assert.Equal(t, "first", response.Checks[0].Name)
	assert.Equal(t, "second", response.Checks[1].Name)
	assert.Equal(t, "third", response.Checks[2].Name)
}

func TestHandlerPanicsForNilRegistry(t *testing.T) {
	assert.Panics(t, func() {
		health.Handler(nil)
	})
}
