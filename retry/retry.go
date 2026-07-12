package retry

import (
	"context"
	"errors"
	"time"
)

// Operation is an error-returning operation that may be retried.
type Operation func(context.Context) error

// ValueOperation is a value-returning operation that may be retried.
type ValueOperation[T any] func(context.Context) (T, error)

// Do executes operation until it succeeds, cannot be retried, exhausts the
// configured attempts, or ctx is canceled.
//
// Do panics if ctx or operation is nil.
func Do(ctx context.Context, operation Operation, options ...Option) error {
	if operation == nil {
		panic("retry: nil operation")
	}
	_, err := DoValue(ctx, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, operation(ctx)
	}, options...)
	return err
}

// DoValue executes operation until it succeeds, cannot be retried, exhausts
// the configured attempts, or ctx is canceled. It returns the value produced
// by the final operation execution, including when that execution fails.
//
// DoValue panics if ctx or operation is nil.
func DoValue[T any](
	ctx context.Context,
	operation ValueOperation[T],
	options ...Option,
) (T, error) {
	if ctx == nil {
		panic("retry: nil context")
	}
	if operation == nil {
		panic("retry: nil operation")
	}

	c := newConfig(options...)
	var result T

	for attempt := 1; attempt <= c.maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return result, err
		}

		var err error
		result, err = operation(ctx)
		if err == nil {
			return result, nil
		}

		info := inspectError(err)
		if info.permanent {
			return result, info.cause
		}
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if attempt == c.maxAttempts || !c.retryIf(info.cause) {
			return result, info.cause
		}

		delay := c.backoff(attempt)
		if delay < 0 {
			panic("retry: backoff returned a negative delay")
		}
		if info.override {
			delay = info.overrideDelay
		}
		if c.notify != nil {
			c.notify(ctx, Event{
				Attempt:     attempt,
				MaxAttempts: c.maxAttempts,
				Err:         info.cause,
				Delay:       delay,
			})
		}
		if err := wait(ctx, delay); err != nil {
			return result, err
		}
	}

	panic("retry: unreachable")
}

// Permanent marks err as non-retryable. It returns nil when err is nil.
//
// The marker supports errors.Is and errors.As through error unwrapping. Do and
// DoValue return the underlying error rather than the marker.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &permanentError{err: err}
}

// After requests delay before the next attempt. It returns nil when err is nil.
//
// The override still respects context cancellation, the retry predicate, and
// the configured attempt limit. After panics if delay is negative.
func After(delay time.Duration, err error) error {
	validateDelay("retry-after delay", delay)
	if err == nil {
		return nil
	}
	return &afterError{delay: delay, err: err}
}

type permanentError struct {
	err error
}

func (e *permanentError) Error() string {
	return e.err.Error()
}

func (e *permanentError) Unwrap() error {
	return e.err
}

type afterError struct {
	delay time.Duration
	err   error
}

func (e *afterError) Error() string {
	return e.err.Error()
}

func (e *afterError) Unwrap() error {
	return e.err
}

type errorInfo struct {
	cause         error
	permanent     bool
	override      bool
	overrideDelay time.Duration
}

func inspectError(err error) errorInfo {
	info := errorInfo{cause: err}

	var permanentErr *permanentError
	if errors.As(err, &permanentErr) {
		info.permanent = true
		info.cause = permanentErr.err
	}

	var afterErr *afterError
	if errors.As(err, &afterErr) {
		info.override = true
		info.overrideDelay = afterErr.delay
		info.cause = afterErr.err
	}

	var nestedPermanent *permanentError
	if errors.As(info.cause, &nestedPermanent) {
		info.permanent = true
		info.cause = nestedPermanent.err
	}

	return info
}

func wait(ctx context.Context, delay time.Duration) error {
	if delay == 0 {
		return ctx.Err()
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
