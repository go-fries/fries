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

// Backend stores and consumes queue tasks with Redis Streams.
type Backend struct {
	redis       goredis.UniversalClient
	prefix      string
	group       string
	consumer    string
	promoteSize int
}

// Option configures a Redis queue backend.
type Option interface {
	apply(*Backend)
}

type optionFunc func(*Backend)

func (f optionFunc) apply(b *Backend) {
	f(b)
}

// WithPrefix sets the Redis key prefix.
func WithPrefix(prefix string) Option {
	return optionFunc(func(b *Backend) {
		if prefix != "" {
			b.prefix = strings.TrimSuffix(prefix, ":")
		}
	})
}

// WithGroup sets the Redis Streams consumer group name.
func WithGroup(group string) Option {
	return optionFunc(func(b *Backend) {
		if group != "" {
			b.group = group
		}
	})
}

// WithConsumer sets the Redis Streams consumer name.
func WithConsumer(consumer string) Option {
	return optionFunc(func(b *Backend) {
		if consumer != "" {
			b.consumer = consumer
		}
	})
}

// WithPromoteSize sets the maximum delayed tasks promoted before each dequeue.
func WithPromoteSize(size int) Option {
	return optionFunc(func(b *Backend) {
		if size > 0 {
			b.promoteSize = size
		}
	})
}

var _ queue.Backend = (*Backend)(nil)

// NewBackend creates a Redis Streams backend.
func NewBackend(redis goredis.UniversalClient, opts ...Option) *Backend {
	b := &Backend{
		redis:       redis,
		prefix:      defaultPrefix,
		group:       defaultGroup,
		consumer:    defaultConsumer,
		promoteSize: defaultPromoteBy,
	}
	for _, opt := range opts {
		opt.apply(b)
	}
	return b
}

// Enqueue stores task in a stream or delayed sorted set.
func (b *Backend) Enqueue(ctx context.Context, task *queue.Task) error {
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
		return b.redis.ZAdd(ctx, b.delayedKey(task.Queue), goredis.Z{
			Score:  float64(task.AvailableAt.UnixNano()),
			Member: string(data),
		}).Err()
	}

	return b.addToStream(ctx, task.Queue, data)
}

// Dequeue returns a task lease from a Redis stream consumer group.
func (b *Backend) Dequeue(ctx context.Context, name string, visibilityTimeout time.Duration) (*queue.Lease, error) {
	if name == "" {
		name = queue.DefaultQueue
	}
	if err := b.ensureGroup(ctx, name); err != nil {
		return nil, err
	}
	if err := b.promoteDue(ctx, name); err != nil {
		return nil, err
	}

	if visibilityTimeout > 0 {
		lease, err := b.claimPending(ctx, name, visibilityTimeout)
		if err != nil {
			return nil, err
		}
		if lease != nil {
			return lease, nil
		}
	}

	streams, err := b.redis.XReadGroup(ctx, &goredis.XReadGroupArgs{
		Group:    b.group,
		Consumer: b.consumer,
		Streams:  []string{b.streamKey(name), ">"},
		Count:    1,
		Block:    -1,
	}).Result()
	if errors.Is(err, goredis.Nil) || len(streams) == 0 || len(streams[0].Messages) == 0 {
		return nil, queue.ErrNoTask
	}
	if err != nil {
		return nil, err
	}

	return b.leaseFromMessage(streams[0].Messages[0])
}

// Ack acknowledges a leased stream message.
func (b *Backend) Ack(ctx context.Context, lease *queue.Lease) error {
	if lease == nil || lease.Task == nil || lease.Token == "" {
		return nil
	}
	return b.redis.XAck(ctx, b.streamKey(lease.Task.Queue), b.group, lease.Token).Err()
}

// Retry re-enqueues a leased task and acknowledges the original stream message.
func (b *Backend) Retry(ctx context.Context, lease *queue.Lease, delay time.Duration) error {
	if lease == nil || lease.Task == nil {
		return nil
	}

	task := lease.Task.Clone()
	task.AvailableAt = time.Now().UTC().Add(delay)
	if err := b.Enqueue(ctx, task); err != nil {
		return err
	}
	return b.Ack(ctx, lease)
}

// DeadLetter writes a leased task to the dead-letter stream and acknowledges the original message.
func (b *Backend) DeadLetter(ctx context.Context, lease *queue.Lease, reason string) error {
	if lease == nil || lease.Task == nil {
		return nil
	}

	data, err := json.Marshal(lease.Task)
	if err != nil {
		return err
	}
	if err := b.redis.XAdd(ctx, &goredis.XAddArgs{
		Stream: b.deadLetterKey(lease.Task.Queue),
		Values: map[string]any{
			taskField:       data,
			deadReasonField: reason,
		},
	}).Err(); err != nil {
		return err
	}
	return b.Ack(ctx, lease)
}

func (b *Backend) claimPending(ctx context.Context, name string, visibilityTimeout time.Duration) (*queue.Lease, error) {
	messages, _, err := b.redis.XAutoClaim(ctx, &goredis.XAutoClaimArgs{
		Stream:   b.streamKey(name),
		Group:    b.group,
		Consumer: b.consumer,
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
	return b.leaseFromMessage(messages[0])
}

func (b *Backend) leaseFromMessage(message goredis.XMessage) (*queue.Lease, error) {
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

func (b *Backend) addToStream(ctx context.Context, name string, data []byte) error {
	return b.redis.XAdd(ctx, &goredis.XAddArgs{
		Stream: b.streamKey(name),
		Values: map[string]any{
			taskField: data,
		},
	}).Err()
}

func (b *Backend) ensureGroup(ctx context.Context, name string) error {
	err := b.redis.XGroupCreateMkStream(ctx, b.streamKey(name), b.group, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

func (b *Backend) promoteDue(ctx context.Context, name string) error {
	return b.redis.Eval(
		ctx, promoteScript, []string{b.delayedKey(name), b.streamKey(name)},
		time.Now().UTC().UnixNano(),
		b.promoteSize,
		taskField,
	).Err()
}

func (b *Backend) streamKey(name string) string {
	return b.prefix + ":" + name + ":stream"
}

func (b *Backend) delayedKey(name string) string {
	return b.prefix + ":" + name + ":delayed"
}

func (b *Backend) deadLetterKey(name string) string {
	return b.prefix + ":" + name + ":dead"
}
