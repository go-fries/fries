package redis

import "strings"

const (
	defaultPrefix    = "queue"
	defaultGroup     = "queue"
	defaultConsumer  = "worker"
	defaultPromoteBy = 100
)

// Option configures a Redis queue.
type Option interface {
	apply(*Queue)
}

type optionFunc func(*Queue)

func (f optionFunc) apply(q *Queue) {
	f(q)
}

// WithPrefix sets the Redis key prefix.
func WithPrefix(prefix string) Option {
	return optionFunc(func(q *Queue) {
		if prefix != "" {
			q.prefix = strings.TrimSuffix(prefix, ":")
		}
	})
}

// WithGroup sets the Redis Streams consumer group name.
func WithGroup(group string) Option {
	return optionFunc(func(q *Queue) {
		if group != "" {
			q.group = group
		}
	})
}

// WithConsumer sets the Redis Streams consumer name.
func WithConsumer(consumer string) Option {
	return optionFunc(func(q *Queue) {
		if consumer != "" {
			q.consumer = consumer
		}
	})
}

// WithPromoteSize sets the maximum delayed tasks promoted before each dequeue.
func WithPromoteSize(size int) Option {
	return optionFunc(func(q *Queue) {
		if size > 0 {
			q.promoteSize = size
		}
	})
}
