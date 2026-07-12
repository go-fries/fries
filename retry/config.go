package retry

import (
	"context"
	"errors"
	"time"
)

const (
	defaultMaxAttempts  = 3
	defaultInitialDelay = 100 * time.Millisecond
	defaultMaximumDelay = time.Second
)

// Event describes a failed operation that will be retried.
type Event struct {
	// Attempt is the one-based number of the execution that failed.
	Attempt int
	// MaxAttempts is the configured total execution limit, including the
	// initial attempt.
	MaxAttempts int
	// Err is the underlying error returned by the failed operation.
	Err error
	// Delay is the duration to wait before the next attempt.
	Delay time.Duration
}

// NotifyFunc receives an Event before the wait for the next attempt begins.
//
// Notifications run synchronously. Implementations should return quickly and
// must be safe for concurrent use when the same function is shared by callers.
type NotifyFunc func(context.Context, Event)

type config struct {
	maxAttempts int
	backoff     Backoff
	retryIf     func(error) bool
	notify      NotifyFunc
}

// Option configures retry execution.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

// WithMaxAttempts sets the total number of allowed executions, including the
// initial attempt.
//
// Values less than one leave the current attempt limit unchanged.
func WithMaxAttempts(attempts int) Option {
	return optionFunc(func(c *config) {
		if attempts >= 1 {
			c.maxAttempts = attempts
		}
	})
}

// WithBackoff sets the delay strategy used between attempts.
//
// A nil backoff leaves the current strategy unchanged.
func WithBackoff(backoff Backoff) Option {
	return optionFunc(func(c *config) {
		if backoff != nil {
			c.backoff = backoff
		}
	})
}

// WithRetryIf sets the predicate that decides whether an operation error may
// be retried. The predicate is not called after the final allowed attempt.
//
// A nil predicate leaves the current predicate unchanged.
func WithRetryIf(predicate func(error) bool) Option {
	return optionFunc(func(c *config) {
		if predicate != nil {
			c.retryIf = predicate
		}
	})
}

// WithNotify sets the function called before each scheduled retry.
//
// Passing a nil [NotifyFunc] disables notifications.
func WithNotify(notify NotifyFunc) Option {
	return optionFunc(func(c *config) {
		c.notify = notify
	})
}

func newConfig(options ...Option) *config {
	c := &config{
		maxAttempts: defaultMaxAttempts,
		backoff:     Exponential(defaultInitialDelay, defaultMaximumDelay),
		retryIf:     defaultRetryIf,
	}
	for _, option := range options {
		if option != nil {
			option.apply(c)
		}
	}
	return c
}

func defaultRetryIf(err error) bool {
	return !errors.Is(err, context.Canceled) &&
		!errors.Is(err, context.DeadlineExceeded)
}
