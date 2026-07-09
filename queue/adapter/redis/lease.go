package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-fries/fries/queue/v4"
	goredis "github.com/redis/go-redis/v9"
)

type redisDelivery struct {
	queue    *Queue
	task     *queue.Task
	streamID string
}

func (d *redisDelivery) Task() *queue.Task {
	if d == nil {
		return nil
	}
	return d.task
}

func (d *redisDelivery) Ack(ctx context.Context) error {
	if d == nil || d.queue == nil || d.task == nil || d.streamID == "" {
		return nil
	}
	return d.queue.redis.XAck(ctx, d.queue.streamKey(d.task.Queue), d.queue.group, d.streamID).Err()
}

func (d *redisDelivery) Retry(ctx context.Context, delay time.Duration) error {
	if d == nil || d.task == nil {
		return nil
	}

	task := d.task.Clone()
	task.AvailableAt = time.Now().UTC().Add(delay)
	if err := d.queue.Enqueue(ctx, task); err != nil {
		return err
	}
	return d.Ack(ctx)
}

func (d *redisDelivery) DeadLetter(ctx context.Context, reason string) error {
	if d == nil || d.task == nil {
		return nil
	}

	data, err := json.Marshal(d.task)
	if err != nil {
		return err
	}
	if err := d.queue.redis.XAdd(ctx, &goredis.XAddArgs{
		Stream: d.queue.deadLetterKey(d.task.Queue),
		MaxLen: d.queue.deadLetterMaxLen,
		Approx: d.queue.deadLetterMaxLen > 0,
		Values: map[string]any{
			taskField:       data,
			deadReasonField: reason,
		},
	}).Err(); err != nil {
		return err
	}
	return d.Ack(ctx)
}
