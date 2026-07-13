package retry

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-fries/fries/hyperf/jet/middleware/timeout/v4"
	"github.com/go-fries/fries/hyperf/jet/v4"
	baseretry "github.com/go-fries/fries/retry/v4"
)

// DefaultRetryIf reports whether err is retryable by the Jet middleware.
//
// It retries timeout middleware errors, HTTP 408 and 429 responses, and HTTP
// 5xx responses. Other errors require an explicit retry predicate.
func DefaultRetryIf(err error) bool {
	if errors.Is(err, timeout.ErrTimeout) {
		return true
	}

	var serverErr *jet.HTTPTransporterServerError
	if !errors.As(err, &serverErr) {
		return false
	}

	return serverErr.StatusCode == http.StatusRequestTimeout ||
		serverErr.StatusCode == http.StatusTooManyRequests ||
		serverErr.StatusCode >= http.StatusInternalServerError &&
			serverErr.StatusCode < 600
}

// New returns Jet middleware that retries calls through the base retry
// component.
//
// The middleware uses [DefaultRetryIf] unless options contain a later
// [baseretry.WithRetryIf] option. Attempt limits, backoff, notifications,
// context cancellation, and final error behavior are owned by the base retry
// component.
func New(options ...baseretry.Option) jet.Middleware {
	retryOptions := make([]baseretry.Option, 0, len(options)+1)
	retryOptions = append(retryOptions, baseretry.WithRetryIf(DefaultRetryIf))
	retryOptions = append(retryOptions, options...)

	return func(next jet.Handler) jet.Handler {
		return func(
			ctx context.Context,
			service string,
			method string,
			request any,
		) (any, error) {
			return baseretry.DoValue(ctx, func(ctx context.Context) (any, error) {
				return next(ctx, service, method, request)
			}, retryOptions...)
		}
	}
}
