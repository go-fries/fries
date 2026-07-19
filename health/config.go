package health

import "time"

const (
	defaultTimeout     = 5 * time.Second
	defaultConcurrency = 4
)

type config struct {
	timeout     time.Duration
	concurrency int
}

// Option configures a [Registry].
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

// WithTimeout sets the total time available for one [Registry.Check].
// Non-positive durations are ignored.
func WithTimeout(timeout time.Duration) Option {
	return optionFunc(func(c *config) {
		if timeout > 0 {
			c.timeout = timeout
		}
	})
}

// WithConcurrency sets the maximum number of checks that may run at once.
// Non-positive limits are ignored.
func WithConcurrency(limit int) Option {
	return optionFunc(func(c *config) {
		if limit > 0 {
			c.concurrency = limit
		}
	})
}

func newConfig(options ...Option) config {
	c := config{
		timeout:     defaultTimeout,
		concurrency: defaultConcurrency,
	}
	for _, option := range options {
		if option != nil {
			option.apply(&c)
		}
	}
	return c
}
