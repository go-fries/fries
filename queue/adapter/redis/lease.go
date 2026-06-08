package redis

import "github.com/go-fries/fries/queue/v3"

type redisLease struct {
	task     *queue.Task
	streamID string
}

func (l *redisLease) Task() *queue.Task {
	if l == nil {
		return nil
	}
	return l.task
}
