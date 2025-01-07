package retry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-fries/fries/hyperf/jet/v3"
)

func New(opts ...Option) jet.Middleware {
	o := options{
		attempts: 3, //nolint:mnd
		backoff:  DefaultBackoff,
		allow:    DefaultAllow,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return func(next jet.Handler) jet.Handler {
		return func(ctx context.Context, service, method string, request any) (response any, err error) {
			starting := time.Now()
			for i := 1; i <= o.attempts; i++ {
				response, err = next(ctx, service, method, request)
				if err == nil {
					return
				}

				if !o.allow(err) {
					return
				}

				if i < o.attempts {
					if sleep := o.backoff(i); sleep > 0 {
						time.Sleep(sleep)
					}
				}
			}
			return response, &Error{
				Attempts: o.attempts,
				Start:    starting,
				End:      time.Now(),
				Err:      err,
			}
		}
	}
}

type Error struct {
	Attempts int
	Start    time.Time
	End      time.Time
	Err      error
}

func (e *Error) Error() string {
	return fmt.Sprintf("jet/middleware/retry: retry failed after %d attempts, time: %v, last error: %v", e.Attempts, e.End.Sub(e.Start), e.Err) //nolint:lll
}

func (e *Error) Unwrap() error {
	return e.Err
}

func IsError(err error) bool {
	var e *Error
	ok := errors.As(err, &e)
	return ok
}
