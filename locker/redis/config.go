package redis

import "time"

const (
	defaultMinWaitInterval = 50 * time.Millisecond
	defaultMaxWaitInterval = 100 * time.Millisecond
)

type config struct {
	minWaitInterval time.Duration
	maxWaitInterval time.Duration
}

// Option configures a [Locker].
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

// WithWaitInterval sets the interval used between acquisition attempts.
// Each interval is chosen randomly from min up to max. Equal values configure
// a fixed interval. Invalid values are ignored.
func WithWaitInterval(minimum, maximum time.Duration) Option {
	return optionFunc(func(c *config) {
		if minimum > 0 && maximum >= minimum {
			c.minWaitInterval = minimum
			c.maxWaitInterval = maximum
		}
	})
}

func newConfig(options ...Option) config {
	c := config{
		minWaitInterval: defaultMinWaitInterval,
		maxWaitInterval: defaultMaxWaitInterval,
	}
	for _, option := range options {
		if option != nil {
			option.apply(&c)
		}
	}
	return c
}
