package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-fries/fries/queue/v3"
	goredis "github.com/redis/go-redis/v9"
)

const (
	taskField       = "task"
	deadReasonField = "reason"
)

// Queue stores and consumes queue tasks with Redis Streams.
type Queue struct {
	redis        goredis.UniversalClient
	prefix       string
	group        string
	consumer     string
	promoteSize  int
	claimMinIdle time.Duration
}

var _ queue.Queue = (*Queue)(nil)

// NewQueue creates a Redis Streams queue.
func NewQueue(redis goredis.UniversalClient, opts ...Option) *Queue {
	c := newConfig(opts...)
	q := &Queue{
		redis:        redis,
		prefix:       c.prefix,
		group:        c.group,
		consumer:     c.consumer,
		promoteSize:  c.promoteSize,
		claimMinIdle: c.claimMinIdle,
	}
	return q
}

// Enqueue stores task in a stream or delayed sorted set.
func (q *Queue) Enqueue(ctx context.Context, task *queue.Task) error {
	if task == nil {
		return nil
	}

	task = task.Clone()
	if task.Queue == "" {
		task.Queue = queue.DefaultQueue
	}

	data, err := json.Marshal(task)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	if task.AvailableAt.After(now) {
		return q.redis.ZAdd(ctx, q.delayedKey(task.Queue), goredis.Z{
			Score:  float64(task.AvailableAt.UnixNano()),
			Member: string(data),
		}).Err()
	}

	return q.addToStream(ctx, task.Queue, data)
}

// Dequeue returns a task lease from a Redis stream consumer group.
func (q *Queue) Dequeue(ctx context.Context, name string) (queue.Lease, error) {
	if name == "" {
		name = queue.DefaultQueue
	}
	if err := q.ensureGroup(ctx, name); err != nil {
		return nil, err
	}
	if err := q.promoteDue(ctx, name); err != nil {
		return nil, err
	}

	if q.claimMinIdle > 0 {
		lease, err := q.claimPending(ctx, name, q.claimMinIdle)
		if err != nil {
			return nil, err
		}
		if lease != nil {
			return lease, nil
		}
	}

	streams, err := q.redis.XReadGroup(ctx, &goredis.XReadGroupArgs{
		Group:    q.group,
		Consumer: q.consumer,
		Streams:  []string{q.streamKey(name), ">"},
		Count:    1,
		Block:    -1,
	}).Result()
	if errors.Is(err, goredis.Nil) {
		return nil, queue.ErrNoTask
	}
	if err != nil {
		return nil, err
	}
	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return nil, queue.ErrNoTask
	}

	return q.leaseFromMessage(streams[0].Messages[0])
}

// Ack acknowledges a leased stream message.
func (q *Queue) Ack(ctx context.Context, lease queue.Lease) error {
	l, ok := lease.(*redisLease)
	if !ok || l == nil || l.task == nil || l.streamID == "" {
		return nil
	}
	return q.redis.XAck(ctx, q.streamKey(l.task.Queue), q.group, l.streamID).Err()
}

// Retry re-enqueues a leased task and acknowledges the original stream message.
func (q *Queue) Retry(ctx context.Context, lease queue.Lease, delay time.Duration) error {
	if lease == nil {
		return nil
	}
	task := lease.Task()
	if task == nil {
		return nil
	}

	task = task.Clone()
	task.AvailableAt = time.Now().UTC().Add(delay)
	if err := q.Enqueue(ctx, task); err != nil {
		return err
	}
	return q.Ack(ctx, lease)
}

// DeadLetter writes a leased task to the dead-letter stream and acknowledges the original message.
func (q *Queue) DeadLetter(ctx context.Context, lease queue.Lease, reason string) error {
	if lease == nil {
		return nil
	}
	task := lease.Task()
	if task == nil {
		return nil
	}

	data, err := json.Marshal(task)
	if err != nil {
		return err
	}
	if err := q.redis.XAdd(ctx, &goredis.XAddArgs{
		Stream: q.deadLetterKey(task.Queue),
		Values: map[string]any{
			taskField:       data,
			deadReasonField: reason,
		},
	}).Err(); err != nil {
		return err
	}
	return q.Ack(ctx, lease)
}
