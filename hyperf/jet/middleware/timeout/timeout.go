package timeout

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-fires/fires/v3/hyperf/jet"
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

func New(opts ...Option) jet.Middleware {
	o := options{
		timeout: defaultTimeout,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return func(next jet.Handler) jet.Handler {
		return func(ctx context.Context, service, method string, request any) (response any, err error) { // nolint:lll
			newCtx, cancel := context.WithTimeout(ctx, o.timeout)
			defer cancel()

			finished := make(chan struct{}, 1)
			mu := sync.Mutex{}

			go func() {
				defer close(finished)
				mu.Lock()
				defer mu.Unlock()
				response, err = next(newCtx, service, method, request)
			}()

			select {
			case <-newCtx.Done():
				mu.Lock()
				defer mu.Unlock()
				if errors.Is(newCtx.Err(), context.DeadlineExceeded) {
					return nil, ErrTimeout
				}
				return nil, newCtx.Err()
			case <-finished:
				mu.Lock()
				defer mu.Unlock()
				return response, err
			}
		}
	}
}
