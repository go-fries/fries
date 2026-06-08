package redis

import "strings"

type config struct {
	prefix      string
	group       string
	consumer    string
	promoteSize int
}

// Option configures a Redis queue.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

// WithPrefix sets the Redis key prefix.
func WithPrefix(prefix string) Option {
	return optionFunc(func(c *config) {
		if prefix != "" {
			c.prefix = strings.TrimSuffix(prefix, ":")
		}
	})
}

// WithGroup sets the Redis Streams consumer group name.
func WithGroup(group string) Option {
	return optionFunc(func(c *config) {
		if group != "" {
			c.group = group
		}
	})
}

// WithConsumer sets the Redis Streams consumer name.
func WithConsumer(consumer string) Option {
	return optionFunc(func(c *config) {
		if consumer != "" {
			c.consumer = consumer
		}
	})
}

// WithPromoteSize sets the maximum delayed tasks promoted before each dequeue.
func WithPromoteSize(size int) Option {
	return optionFunc(func(c *config) {
		if size > 0 {
			c.promoteSize = size
		}
	})
}

func newConfig(opts ...Option) *config {
	c := &config{
		prefix:      "queue",
		group:       "queue",
		consumer:    "worker",
		promoteSize: 100,
	}
	for _, opt := range opts {
		opt.apply(c)
	}
	return c
}
