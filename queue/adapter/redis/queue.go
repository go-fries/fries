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
	receiveBlock    = time.Second
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

// NewConsumer creates a Redis Streams consumer using config.
func (q *Queue) NewConsumer(ctx context.Context, config queue.ConsumerConfig) (queue.Consumer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	config = config.Normalize()
	consumerName := config.Name
	if consumerName == "" {
		consumerName = q.consumer
	}
	if err := q.ensureGroup(ctx, config.Queue); err != nil {
		return nil, err
	}

	consumerCtx, cancel := context.WithCancel(context.Background())
	return &consumer{
		queue:        q,
		name:         config.Queue,
		consumerName: consumerName,
		ctx:          consumerCtx,
		cancel:       cancel,
	}, nil
}

type consumer struct {
	queue        *Queue
	name         string
	consumerName string
	ctx          context.Context
	cancel       context.CancelFunc
}

func (c *consumer) Receive(ctx context.Context) (queue.Delivery, error) {
	receiveCtx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(c.ctx, cancel)
	defer func() {
		stop()
		cancel()
	}()

	for {
		delivery, err := c.queue.receiveForConsumer(receiveCtx, c.name, c.consumerName)
		if err != nil && errors.Is(err, context.Canceled) && ctx.Err() == nil && c.ctx.Err() != nil {
			return nil, queue.ErrConsumerClosed
		}
		if errors.Is(err, queue.ErrNoTask) {
			continue
		}
		return delivery, err
	}
}

func (c *consumer) Close() error {
	c.cancel()
	return nil
}

func (q *Queue) receive(ctx context.Context, name string) (queue.Delivery, error) {
	return q.receiveForConsumer(ctx, name, q.consumer)
}

func (q *Queue) receiveForConsumer(ctx context.Context, name, consumerName string) (queue.Delivery, error) {
	if err := q.promoteDue(ctx, name); err != nil {
		return nil, err
	}

	if q.claimMinIdle > 0 {
		delivery, err := q.claimPendingForConsumer(ctx, name, consumerName, q.claimMinIdle)
		if err != nil {
			return nil, err
		}
		if delivery != nil {
			return delivery, nil
		}
	}

	streams, err := q.redis.XReadGroup(ctx, &goredis.XReadGroupArgs{
		Group:    q.group,
		Consumer: consumerName,
		Streams:  []string{q.streamKey(name), ">"},
		Count:    1,
		Block:    receiveBlock,
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
