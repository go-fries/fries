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
	Attempt     int
	MaxAttempts int
	Err         error
	Delay       time.Duration
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
// It panics if attempts is less than one.
func WithMaxAttempts(attempts int) Option {
	if attempts < 1 {
		panic("retry: max attempts must be greater than zero")
	}
	return optionFunc(func(c *config) {
		c.maxAttempts = attempts
	})
}

// WithBackoff sets the delay strategy used between attempts.
//
// It panics if backoff is nil.
func WithBackoff(backoff Backoff) Option {
	if backoff == nil {
		panic("retry: nil backoff")
	}
	return optionFunc(func(c *config) {
		c.backoff = backoff
	})
}

// WithRetryIf sets the predicate that decides whether an operation error may
// be retried. The predicate is not called after the final allowed attempt.
//
// It panics if predicate is nil.
func WithRetryIf(predicate func(error) bool) Option {
	if predicate == nil {
		panic("retry: nil retry predicate")
	}
	return optionFunc(func(c *config) {
		c.retryIf = predicate
	})
}

// WithNotify sets the function called before each scheduled retry.
//
// Passing nil disables notifications.
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
