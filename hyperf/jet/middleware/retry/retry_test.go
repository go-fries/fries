package retry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-fries/fries/hyperf/jet/middleware/timeout/v4"
	"github.com/go-fries/fries/hyperf/jet/v4"
	baseretry "github.com/go-fries/fries/retry/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultRetryIf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "timeout", err: timeout.ErrTimeout, want: true},
		{name: "wrapped timeout", err: fmt.Errorf("call: %w", timeout.ErrTimeout), want: true},
		{name: "request timeout", err: serverError(http.StatusRequestTimeout), want: true},
		{name: "too many requests", err: serverError(http.StatusTooManyRequests), want: true},
		{name: "internal server error", err: serverError(http.StatusInternalServerError), want: true},
		{name: "service unavailable", err: serverError(http.StatusServiceUnavailable), want: true},
		{name: "wrapped service unavailable", err: fmt.Errorf("call: %w", serverError(http.StatusServiceUnavailable)), want: true},
		{name: "bad request", err: serverError(http.StatusBadRequest), want: false},
		{name: "not found", err: serverError(http.StatusNotFound), want: false},
		{name: "nonstandard 6xx", err: serverError(600), want: false},
		{name: "unrelated", err: assert.AnError, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, DefaultRetryIf(tt.err))
		})
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("retries default retryable errors", func(t *testing.T) {
		t.Parallel()
		var attempts int
		handler := New(baseretry.WithBackoff(baseretry.NoBackoff()))(
			func(context.Context, string, string, any) (any, error) {
				attempts++
				if attempts < 3 {
					return nil, timeout.ErrTimeout
				}
				return "ok", nil
			},
		)

		response, err := handler(t.Context(), "service", "method", "request")
		require.NoError(t, err)
		assert.Equal(t, "ok", response)
		assert.Equal(t, 3, attempts)
	})

	t.Run("does not retry non-retryable errors", func(t *testing.T) {
		t.Parallel()
		var attempts int
		handler := New(baseretry.WithBackoff(baseretry.NoBackoff()))(
			func(context.Context, string, string, any) (any, error) {
				attempts++
				return "partial", assert.AnError
			},
		)

		response, err := handler(t.Context(), "service", "method", nil)
		require.ErrorIs(t, err, assert.AnError)
		assert.Equal(t, "partial", response)
		assert.Equal(t, 1, attempts)
	})

	t.Run("custom predicate overrides default", func(t *testing.T) {
		t.Parallel()
		customErr := errors.New("temporary business error")
		var attempts int
		handler := New(
			baseretry.WithMaxAttempts(2),
			baseretry.WithBackoff(baseretry.NoBackoff()),
			baseretry.WithRetryIf(func(err error) bool {
				return errors.Is(err, customErr)
			}),
		)(func(context.Context, string, string, any) (any, error) {
			attempts++
			return nil, customErr
		})

		_, err := handler(t.Context(), "service", "method", nil)
		require.ErrorIs(t, err, customErr)
		assert.Equal(t, customErr, err)
		assert.Equal(t, 2, attempts)
	})

	t.Run("context cancellation stops retries", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		var attempts int
		handler := New(baseretry.WithBackoff(baseretry.NoBackoff()))(
			func(context.Context, string, string, any) (any, error) {
				attempts++
				cancel()
				return nil, timeout.ErrTimeout
			},
		)

		_, err := handler(ctx, "service", "method", nil)
		require.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, 1, attempts)
	})

	t.Run("copies options", func(t *testing.T) {
		t.Parallel()
		options := []baseretry.Option{
			baseretry.WithMaxAttempts(2),
			baseretry.WithBackoff(baseretry.NoBackoff()),
		}
		middleware := New(options...)
		options[0] = baseretry.WithMaxAttempts(5)

		var attempts int
		handler := middleware(func(context.Context, string, string, any) (any, error) {
			attempts++
			return nil, timeout.ErrTimeout
		})

		_, err := handler(t.Context(), "service", "method", nil)
		require.ErrorIs(t, err, timeout.ErrTimeout)
		assert.Equal(t, 2, attempts)
	})
}

func serverError(statusCode int) error {
	return &jet.HTTPTransporterServerError{
		StatusCode: statusCode,
		Message:    http.StatusText(statusCode),
		Err:        assert.AnError,
	}
}
