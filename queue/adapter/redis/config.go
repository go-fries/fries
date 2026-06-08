package redis

import (
	"strings"
	"time"
)

type config struct {
	prefix       string
	group        string
	consumer     string
	promoteSize  int
	claimMinIdle time.Duration
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

// WithPromoteSize sets the maximum delayed tasks promoted before each receive attempt.
func WithPromoteSize(size int) Option {
	return optionFunc(func(c *config) {
		if size > 0 {
			c.promoteSize = size
		}
	})
}

// WithClaimMinIdle sets how long a pending stream message must remain idle before a consumer can claim it.
//
// Set minIdle to 0 to disable pending message claims during receive.
func WithClaimMinIdle(minIdle time.Duration) Option {
	return optionFunc(func(c *config) {
		if minIdle >= 0 {
			c.claimMinIdle = minIdle
		}
	})
}

func newConfig(opts ...Option) *config {
	c := &config{
		prefix:       "queue",
		group:        "queue",
		consumer:     "worker",
		promoteSize:  100,
		claimMinIdle: 5 * time.Minute,
	}
	for _, opt := range opts {
		opt.apply(c)
	}
	return c
}
