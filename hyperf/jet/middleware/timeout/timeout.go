package timeout

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-fries/fries/hyperf/jet/v4"
)

var (
	ErrTimeout     = fmt.Errorf("jet/timeout: request timeout")
	defaultTimeout = time.Second * 5
)

type options struct {
	timeout time.Duration
}

type Option func(*options)

func Timeout(timeout time.Duration) Option {
	return func(o *options) {
		o.timeout = timeout
	}
}

// New limits the time spent waiting for a handler. Deadline expiration returns
// ErrTimeout; other context cancellation returns the context error. A handler
// that ignores cancellation may continue running after the middleware returns.
func New(opts ...Option) jet.Middleware {
	o := options{
		timeout: defaultTimeout,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return func(next jet.Handler) jet.Handler {
		return func(ctx context.Context, service, method string, request any) (any, error) {
			newCtx, cancel := context.WithTimeout(ctx, o.timeout)
			defer cancel()

			type result struct {
				response any
				err      error
			}
			finished := make(chan result, 1)

			go func() {
				response, err := next(newCtx, service, method, request)
				finished <- result{response: response, err: err}
			}()

			select {
			case <-newCtx.Done():
				if errors.Is(newCtx.Err(), context.DeadlineExceeded) {
					return nil, ErrTimeout
				}
				return nil, newCtx.Err()
			case result := <-finished:
				return result.response, result.err
			}
		}
	}
}
