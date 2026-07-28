package redis

import "strings"

const defaultPrefix = "fries:idempotency:"

type config struct {
	prefix string
}

// Option configures a [Store].
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

// WithPrefix sets the Redis key prefix. The default is
// "fries:idempotency:". Trailing colons are normalized, and an empty prefix is
// ignored.
func WithPrefix(prefix string) Option {
	return optionFunc(func(c *config) {
		if prefix = strings.TrimRight(prefix, ":"); prefix != "" {
			c.prefix = prefix + ":"
		}
	})
}

func newConfig(options ...Option) config {
	c := config{prefix: defaultPrefix}
	for _, option := range options {
		if option != nil {
			option.apply(&c)
		}
	}
	return c
}
