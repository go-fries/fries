package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-fries/fries/queue/v3"
	goredis "github.com/redis/go-redis/v9"
)

const (
	taskField        = "task"
	deadReasonField  = "reason"
	defaultPrefix    = "queue"
	defaultGroup     = "queue"
	defaultConsumer  = "worker"
	defaultPromoteBy = 100
)

const promoteScript = `
local tasks = redis.call("zrangebyscore", KEYS[1], "-inf", ARGV[1], "limit", 0, ARGV[2])
for _, task in ipairs(tasks) do
	redis.call("xadd", KEYS[2], "*", ARGV[3], task)
	redis.call("zrem", KEYS[1], task)
end
return #tasks
`

// Queue stores and consumes queue tasks with Redis Streams.
type Queue struct {
	redis       goredis.UniversalClient
	prefix      string
	group       string
	consumer    string
	promoteSize int
}

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

var _ queue.Queue = (*Queue)(nil)

// NewQueue creates a Redis Streams queue.
func NewQueue(redis goredis.UniversalClient, opts ...Option) *Queue {
	q := &Queue{
		redis:       redis,
		prefix:      defaultPrefix,
		group:       defaultGroup,
		consumer:    defaultConsumer,
		promoteSize: defaultPromoteBy,
	}
	for _, opt := range opts {
		opt.apply(q)
	}
	return q
}

// Enqueue stores task in a stream or delayed sorted set.
func (q *Queue) Enqueue(ctx context.Context, task *queue.Task) error {
	if task == nil {
		return nil
	}
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
func (q *Queue) Dequeue(ctx context.Context, name string, visibilityTimeout time.Duration) (*queue.Lease, error) {
	if name == "" {
		name = queue.DefaultQueue
	}
	if err := q.ensureGroup(ctx, name); err != nil {
		return nil, err
	}
	if err := q.promoteDue(ctx, name); err != nil {
		return nil, err
	}

	if visibilityTimeout > 0 {
		lease, err := q.claimPending(ctx, name, visibilityTimeout)
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
	if errors.Is(err, goredis.Nil) || len(streams) == 0 || len(streams[0].Messages) == 0 {
		return nil, queue.ErrNoTask
	}
	if err != nil {
		return nil, err
	}

	return q.leaseFromMessage(streams[0].Messages[0])
}

// Ack acknowledges a leased stream message.
func (q *Queue) Ack(ctx context.Context, lease *queue.Lease) error {
	if lease == nil || lease.Task == nil || lease.Token == "" {
		return nil
	}
	return q.redis.XAck(ctx, q.streamKey(lease.Task.Queue), q.group, lease.Token).Err()
}

// Retry re-enqueues a leased task and acknowledges the original stream message.
func (q *Queue) Retry(ctx context.Context, lease *queue.Lease, delay time.Duration) error {
	if lease == nil || lease.Task == nil {
		return nil
	}

	task := lease.Task.Clone()
	task.AvailableAt = time.Now().UTC().Add(delay)
	if err := q.Enqueue(ctx, task); err != nil {
		return err
	}
	return q.Ack(ctx, lease)
}

// DeadLetter writes a leased task to the dead-letter stream and acknowledges the original message.
func (q *Queue) DeadLetter(ctx context.Context, lease *queue.Lease, reason string) error {
	if lease == nil || lease.Task == nil {
		return nil
	}

	data, err := json.Marshal(lease.Task)
	if err != nil {
		return err
	}
	if err := q.redis.XAdd(ctx, &goredis.XAddArgs{
		Stream: q.deadLetterKey(lease.Task.Queue),
		Values: map[string]any{
			taskField:       data,
			deadReasonField: reason,
		},
	}).Err(); err != nil {
		return err
	}
	return q.Ack(ctx, lease)
}

func (q *Queue) claimPending(ctx context.Context, name string, visibilityTimeout time.Duration) (*queue.Lease, error) {
	messages, _, err := q.redis.XAutoClaim(ctx, &goredis.XAutoClaimArgs{
		Stream:   q.streamKey(name),
		Group:    q.group,
		Consumer: q.consumer,
		MinIdle:  visibilityTimeout,
		Start:    "0-0",
		Count:    1,
	}).Result()
	if errors.Is(err, goredis.Nil) || len(messages) == 0 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return q.leaseFromMessage(messages[0])
}

func (q *Queue) leaseFromMessage(message goredis.XMessage) (*queue.Lease, error) {
	value, ok := message.Values[taskField]
	if !ok {
		return nil, fmt.Errorf("queue/redis: message %s missing %q field", message.ID, taskField)
	}

	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return nil, fmt.Errorf("queue/redis: message %s has unsupported %q field", message.ID, taskField)
	}

	var task queue.Task
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, err
	}
	task.Attempt++
	return &queue.Lease{
		Task:  &task,
		Token: message.ID,
	}, nil
}

func (q *Queue) addToStream(ctx context.Context, name string, data []byte) error {
	return q.redis.XAdd(ctx, &goredis.XAddArgs{
		Stream: q.streamKey(name),
		Values: map[string]any{
			taskField: data,
		},
	}).Err()
}

func (q *Queue) ensureGroup(ctx context.Context, name string) error {
	err := q.redis.XGroupCreateMkStream(ctx, q.streamKey(name), q.group, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

func (q *Queue) promoteDue(ctx context.Context, name string) error {
	return q.redis.Eval(
		ctx, promoteScript, []string{q.delayedKey(name), q.streamKey(name)},
		time.Now().UTC().UnixNano(),
		q.promoteSize,
		taskField,
	).Err()
}

func (q *Queue) streamKey(name string) string {
	return q.prefix + ":" + name + ":stream"
}

func (q *Queue) delayedKey(name string) string {
	return q.prefix + ":" + name + ":delayed"
}

func (q *Queue) deadLetterKey(name string) string {
	return q.prefix + ":" + name + ":dead"
}
