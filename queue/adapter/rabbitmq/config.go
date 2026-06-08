package rabbitmq

import (
	"strings"
	"time"
)

const defaultDelayQueueTTL = time.Hour

type config struct {
	prefix        string
	durable       bool
	delayQueueTTL time.Duration
}

// Option configures a RabbitMQ queue adapter.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

// WithPrefix sets the queue name prefix used for ready, delay, and dead-letter queues.
func WithPrefix(prefix string) Option {
	return optionFunc(func(c *config) {
		if prefix != "" {
			c.prefix = strings.TrimSuffix(prefix, ".")
		}
	})
}

// WithDurable sets whether declared RabbitMQ queues survive broker restarts.
func WithDurable(durable bool) Option {
	return optionFunc(func(c *config) {
		c.durable = durable
	})
}

// WithDelayQueueTTL sets how long unused delay queues remain after their delay expires.
func WithDelayQueueTTL(ttl time.Duration) Option {
	return optionFunc(func(c *config) {
		if ttl > 0 {
			c.delayQueueTTL = ttl
		}
	})
}

func newConfig(opts ...Option) *config {
	c := &config{
		durable:       true,
		delayQueueTTL: defaultDelayQueueTTL,
	}
	for _, opt := range opts {
		opt.apply(c)
	}
	return c
}
