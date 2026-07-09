package redis

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-fries/fries/queue/v4"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type redisClientStub struct {
	goredis.UniversalClient

	xAutoClaimMessages []goredis.XMessage
	xAutoClaimErr      error
	xClaimMessages     []goredis.XMessage
	xClaimErr          error
	xPendingExt        []goredis.XPendingExt
	xPendingExtErr     error
}

func (c *redisClientStub) XAutoClaim(ctx context.Context, _ *goredis.XAutoClaimArgs) *goredis.XAutoClaimCmd {
	cmd := goredis.NewXAutoClaimCmd(ctx)
	cmd.SetVal(c.xAutoClaimMessages, "0-0")
	cmd.SetErr(c.xAutoClaimErr)
	return cmd
}

func (c *redisClientStub) XClaim(ctx context.Context, _ *goredis.XClaimArgs) *goredis.XMessageSliceCmd {
	cmd := goredis.NewXMessageSliceCmd(ctx)
	cmd.SetVal(c.xClaimMessages)
	cmd.SetErr(c.xClaimErr)
	return cmd
}

func (c *redisClientStub) XPendingExt(ctx context.Context, _ *goredis.XPendingExtArgs) *goredis.XPendingExtCmd {
	cmd := goredis.NewXPendingExtCmd(ctx)
	cmd.SetVal(c.xPendingExt)
	cmd.SetErr(c.xPendingExtErr)
	return cmd
}

func TestQueue_DeliveryFromMessage(t *testing.T) {
	t.Parallel()

	q := NewQueue(nil)
	task := &queue.Task{
		ID:      "task-1",
		Type:    "send_email",
		Queue:   "default",
		Payload: []byte("hello"),
		Attempt: 2,
	}
	data, err := json.Marshal(task)
	require.NoError(t, err)

	delivery, err := q.leaseFromMessage(goredis.XMessage{
		ID: "1-0",
		Values: map[string]any{
			taskField: string(data),
		},
	})
	require.NoError(t, err)
	require.NotNil(t, delivery)
	require.NotNil(t, delivery.Task())

	l, ok := delivery.(*redisDelivery)
	require.True(t, ok)
	assert.Equal(t, "1-0", l.streamID)
	assert.Equal(t, 3, delivery.Task().Attempt)
	assert.Equal(t, "hello", string(delivery.Task().Payload))
}

func TestQueue_DeliveryFromMessageAcceptsBytes(t *testing.T) {
	t.Parallel()

	q := NewQueue(nil)
	data, err := json.Marshal(&queue.Task{ID: "task-1", Type: "send_email"})
	require.NoError(t, err)

	delivery, err := q.leaseFromMessage(goredis.XMessage{
		ID: "1-0",
		Values: map[string]any{
			taskField: data,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, delivery)
	require.NotNil(t, delivery.Task())
	assert.Equal(t, "task-1", delivery.Task().ID)
	assert.Equal(t, 1, delivery.Task().Attempt)
}

func TestQueue_DeliveryFromMessageWithDeliveryCount(t *testing.T) {
	t.Parallel()

	q := NewQueue(nil)
	data, err := json.Marshal(&queue.Task{ID: "task-1", Type: "send_email", Attempt: 1})
	require.NoError(t, err)

	delivery, err := q.leaseFromMessageWithDeliveryCount(goredis.XMessage{
		ID: "1-0",
		Values: map[string]any{
			taskField: data,
		},
	}, 2)
	require.NoError(t, err)
	require.NotNil(t, delivery)
	require.NotNil(t, delivery.Task())
	assert.Equal(t, 3, delivery.Task().Attempt)
}

func TestAttemptWithDeliveryCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		baseAttempt   int
		deliveryCount int64
		want          int
	}{
		{
			name:          "increments when delivery count is unavailable",
			baseAttempt:   1,
			deliveryCount: 0,
			want:          2,
		},
		{
			name:          "adds redis delivery count to stored attempt",
			baseAttempt:   1,
			deliveryCount: 2,
			want:          3,
		},
		{
			name:          "saturates unavailable delivery count",
			baseAttempt:   math.MaxInt,
			deliveryCount: 0,
			want:          math.MaxInt,
		},
		{
			name:          "saturates large delivery count",
			baseAttempt:   math.MaxInt - 1,
			deliveryCount: 2,
			want:          math.MaxInt,
		},
		{
			name:          "normalizes negative stored attempt",
			baseAttempt:   -1,
			deliveryCount: 2,
			want:          2,
		},
		{
			name:          "normalizes negative stored attempt without delivery count",
			baseAttempt:   -1,
			deliveryCount: 0,
			want:          1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := attemptWithDeliveryCount(tt.baseAttempt, tt.deliveryCount)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestQueue_ClaimPendingReturnsXAutoClaimError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("xautoclaim failed")
	q := NewQueue(&redisClientStub{xAutoClaimErr: wantErr})

	_, err := q.claimPendingForConsumer(t.Context(), "critical", q.consumer, time.Second)

	require.ErrorIs(t, err, wantErr)
}

func TestQueue_ClaimPendingFallsBackToXClaim(t *testing.T) {
	t.Parallel()

	data, err := json.Marshal(&queue.Task{ID: "task-1", Type: "send_email"})
	require.NoError(t, err)
	client := &redisClientStub{
		xPendingExt: []goredis.XPendingExt{
			{ID: "1-0", Idle: time.Second, RetryCount: 2},
		},
		xClaimMessages: []goredis.XMessage{
			{
				ID: "1-0",
				Values: map[string]any{
					taskField: string(data),
				},
			},
		},
	}
	q := NewQueue(client)

	delivery, err := q.claimPendingForConsumer(t.Context(), "critical", q.consumer, time.Second)

	require.NoError(t, err)
	require.NotNil(t, delivery)
	require.NotNil(t, delivery.Task())
	assert.Equal(t, "task-1", delivery.Task().ID)
	assert.Equal(t, 2, delivery.Task().Attempt)
}

func TestQueue_ClaimPendingReturnsXClaimError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("xclaim failed")
	q := NewQueue(&redisClientStub{
		xPendingExt: []goredis.XPendingExt{
			{ID: "1-0", Idle: time.Second},
		},
		xClaimErr: wantErr,
	})

	_, err := q.claimPendingForConsumer(t.Context(), "critical", q.consumer, time.Second)

	require.ErrorIs(t, err, wantErr)
}

func TestQueue_DeliveryCountReturnsXPendingExtError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("xpending failed")
	q := NewQueue(&redisClientStub{xPendingExtErr: wantErr})

	_, err := q.deliveryCount(t.Context(), "critical", "1-0")

	require.ErrorIs(t, err, wantErr)
}

func TestQueue_DeliveryFromMessageErrors(t *testing.T) {
	t.Parallel()

	q := NewQueue(nil)
	tests := []struct {
		name    string
		message goredis.XMessage
		want    string
	}{
		{
			name:    "missing task field",
			message: goredis.XMessage{ID: "1-0", Values: map[string]any{}},
			want:    `missing "task" field`,
		},
		{
			name: "unsupported task field",
			message: goredis.XMessage{
				ID: "1-0",
				Values: map[string]any{
					taskField: 1,
				},
			},
			want: `unsupported "task" field`,
		},
		{
			name: "invalid task json",
			message: goredis.XMessage{
				ID: "1-0",
				Values: map[string]any{
					taskField: "{",
				},
			},
			want: "unexpected end of JSON input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := q.leaseFromMessage(tt.message)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestQueue_OptionsUseDefaultsAndIgnoreInvalidValues(t *testing.T) {
	t.Parallel()

	q := NewQueue(
		nil,
		WithPrefix(""),
		WithGroup(""),
		WithConsumer(""),
		WithPromoteSize(0),
		WithClaimMinIdle(-time.Second),
		WithDeadLetterMaxLen(0),
	)

	assert.Equal(t, "queue:critical:stream", q.streamKey("critical"))
	assert.Equal(t, "queue:critical:delayed", q.delayedKey("critical"))
	assert.Equal(t, "queue:critical:dead", q.deadLetterKey("critical"))
	assert.Equal(t, "queue", q.group)
	assert.NotEmpty(t, q.consumer)
	assert.NotEqual(t, "worker", q.consumer)
	assert.True(t, strings.HasPrefix(q.consumer, "worker-"))
	assert.Equal(t, 100, q.promoteSize)
	assert.Equal(t, 5*time.Minute, q.claimMinIdle)
	assert.Zero(t, q.deadLetterMaxLen)
}

func TestQueue_Options(t *testing.T) {
	t.Parallel()

	q := NewQueue(
		nil,
		WithPrefix("app:"),
		WithGroup("workers"),
		WithConsumer("worker-1"),
		WithPromoteSize(10),
		WithClaimMinIdle(30*time.Second),
		WithDeadLetterMaxLen(100),
	)

	assert.Equal(t, "app:critical:stream", q.streamKey("critical"))
	assert.Equal(t, "app:critical:delayed", q.delayedKey("critical"))
	assert.Equal(t, "app:critical:dead", q.deadLetterKey("critical"))
	assert.Equal(t, "workers", q.group)
	assert.Equal(t, "worker-1", q.consumer)
	assert.Equal(t, 10, q.promoteSize)
	assert.Equal(t, 30*time.Second, q.claimMinIdle)
	assert.Equal(t, int64(100), q.deadLetterMaxLen)
}

func TestQueue_NoopOperations(t *testing.T) {
	t.Parallel()

	q := NewQueue(nil)
	var nilDelivery *redisDelivery

	require.NoError(t, q.Enqueue(t.Context(), nil))
	require.NoError(t, nilDelivery.Ack(t.Context()))
	require.NoError(t, nilDelivery.Retry(t.Context(), 0))
	require.NoError(t, nilDelivery.DeadLetter(t.Context(), "failed"))
	require.NoError(t, (&redisDelivery{queue: q}).Ack(t.Context()))
	require.NoError(t, (&redisDelivery{queue: q}).Retry(t.Context(), 0))
	require.NoError(t, (&redisDelivery{queue: q}).DeadLetter(t.Context(), "failed"))
}

func TestRedisDelivery_TaskNilReceiver(t *testing.T) {
	t.Parallel()

	var delivery *redisDelivery

	assert.Nil(t, delivery.Task())
}

func TestQueue_EnqueueReceiveAck(t *testing.T) {
	t.Parallel()

	q, _ := newRedisTestQueue(t)
	ctx := t.Context()

	_, err := queue.NewProducer(q).Enqueue(ctx, "send_email", []byte("hello"), queue.WithQueue("critical"))
	require.NoError(t, err)

	delivery, err := receive(ctx, q, "critical")
	require.NoError(t, err)
	require.NotNil(t, delivery)
	require.NotNil(t, delivery.Task())
	assert.Equal(t, "send_email", delivery.Task().Type)
	assert.Equal(t, "critical", delivery.Task().Queue)
	assert.Equal(t, []byte("hello"), delivery.Task().Payload)
	assert.Equal(t, 1, delivery.Task().Attempt)

	require.NoError(t, delivery.Ack(ctx))

	_, err = receiveCriticalWithTimeout(ctx, q, 20*time.Millisecond)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestQueue_EnqueueDoesNotMutateTask(t *testing.T) {
	t.Parallel()

	q, _ := newRedisTestQueue(t)
	ctx := t.Context()
	task := &queue.Task{
		ID:      "task-1",
		Type:    "send_email",
		Payload: []byte("hello"),
	}

	require.NoError(t, q.Enqueue(ctx, task))

	assert.Empty(t, task.Queue)

	delivery, err := receive(ctx, q, queue.DefaultQueue)
	require.NoError(t, err)
	require.NotNil(t, delivery)
	require.NotNil(t, delivery.Task())
	assert.Equal(t, queue.DefaultQueue, delivery.Task().Queue)
}

func TestQueue_DelayedTasksWithSamePayloadAreNotCollapsed(t *testing.T) {
	t.Parallel()

	q, client := newRedisTestQueue(t)
	ctx := t.Context()
	task := &queue.Task{
		ID:          "task-1",
		Type:        "send_email",
		Queue:       "critical",
		Payload:     []byte("hello"),
		CreatedAt:   time.Unix(1, 0).UTC(),
		AvailableAt: time.Now().UTC().Add(20 * time.Millisecond),
	}

	require.NoError(t, q.Enqueue(ctx, task))
	require.NoError(t, q.Enqueue(ctx, task))

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.NoError(c, q.promoteDue(ctx, "critical"))

		length, err := client.XLen(ctx, q.streamKey("critical")).Result()
		if err != nil {
			assert.NoError(c, err)
			return
		}
		assert.Equal(c, int64(2), length)
	}, time.Second, 10*time.Millisecond)

	messages, err := client.XRange(ctx, q.streamKey("critical"), "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, messages, 2)
	assert.Equal(t, "send_email", taskFromMessage(t, messages[0]).Type)
	assert.Equal(t, "send_email", taskFromMessage(t, messages[1]).Type)
}

func TestQueue_RetryReenqueuesAndAcksDelivery(t *testing.T) {
	t.Parallel()

	q, _ := newRedisTestQueue(t)
	ctx := t.Context()

	_, err := queue.NewProducer(q).Enqueue(ctx, "send_email", []byte("hello"), queue.WithQueue("critical"))
	require.NoError(t, err)

	delivery, err := receive(ctx, q, "critical")
	require.NoError(t, err)
	require.NoError(t, delivery.Retry(ctx, 0))

	delivery, err = receive(ctx, q, "critical")
	require.NoError(t, err)
	require.NotNil(t, delivery)
	require.NotNil(t, delivery.Task())
	assert.Equal(t, "send_email", delivery.Task().Type)
	assert.Equal(t, 2, delivery.Task().Attempt)
}

func TestQueue_ClaimPendingIncrementsAttempt(t *testing.T) {
	t.Parallel()

	q, client := newRedisTestQueue(t, WithClaimMinIdle(time.Millisecond))
	ctx := t.Context()

	_, err := queue.NewProducer(q).Enqueue(ctx, "send_email", []byte("hello"), queue.WithQueue("critical"))
	require.NoError(t, err)

	delivery, err := receive(ctx, q, "critical")
	require.NoError(t, err)
	require.NotNil(t, delivery)
	require.NotNil(t, delivery.Task())
	assert.Equal(t, 1, delivery.Task().Attempt)

	messages, err := client.XRange(ctx, q.streamKey("critical"), "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, messages, 1)
	stored := taskFromMessage(t, messages[0])
	assert.Equal(t, 0, stored.Attempt)

	claimed, err := q.claimPendingForConsumer(ctx, "critical", "worker-2", 0)
	require.NoError(t, err)

	require.NotNil(t, claimed)
	require.NotNil(t, claimed.Task())
	assert.Equal(t, "send_email", claimed.Task().Type)
	assert.Equal(t, 2, claimed.Task().Attempt)
	require.NoError(t, claimed.Ack(ctx))
}

func TestQueue_ClaimRetriedPendingIncrementsAttempt(t *testing.T) {
	t.Parallel()

	q, client := newRedisTestQueue(t, WithClaimMinIdle(time.Millisecond))
	ctx := t.Context()

	_, err := queue.NewProducer(q).Enqueue(ctx, "send_email", []byte("hello"), queue.WithQueue("critical"))
	require.NoError(t, err)

	delivery, err := receive(ctx, q, "critical")
	require.NoError(t, err)
	require.NotNil(t, delivery)
	require.NotNil(t, delivery.Task())
	require.Equal(t, 1, delivery.Task().Attempt)
	require.NoError(t, delivery.Retry(ctx, 0))

	delivery, err = receive(ctx, q, "critical")
	require.NoError(t, err)
	require.NotNil(t, delivery)
	require.NotNil(t, delivery.Task())
	assert.Equal(t, 2, delivery.Task().Attempt)

	messages, err := client.XRange(ctx, q.streamKey("critical"), "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, messages, 2)
	stored := taskFromMessage(t, messages[1])
	assert.Equal(t, 1, stored.Attempt)

	claimed, err := q.claimPendingForConsumer(ctx, "critical", "worker-2", 0)
	require.NoError(t, err)

	require.NotNil(t, claimed)
	require.NotNil(t, claimed.Task())
	assert.Equal(t, "send_email", claimed.Task().Type)
	assert.Equal(t, 3, claimed.Task().Attempt)
	require.NoError(t, claimed.Ack(ctx))
}

func TestQueue_DeadLetterWritesReasonAndAcksDelivery(t *testing.T) {
	t.Parallel()

	q, client := newRedisTestQueue(t)
	ctx := t.Context()

	_, err := queue.NewProducer(q).Enqueue(ctx, "send_email", []byte("hello"), queue.WithQueue("critical"))
	require.NoError(t, err)

	delivery, err := receive(ctx, q, "critical")
	require.NoError(t, err)
	require.NoError(t, delivery.DeadLetter(ctx, "failed"))

	messages, err := client.XRange(ctx, q.deadLetterKey("critical"), "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Equal(t, "failed", messages[0].Values[deadReasonField])

	_, err = receiveCriticalWithTimeout(ctx, q, 20*time.Millisecond)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestQueue_DelayedTaskPromotion(t *testing.T) {
	t.Parallel()

	q, _ := newRedisTestQueue(t)
	ctx := t.Context()

	_, err := queue.NewProducer(q).Enqueue(ctx, "send_email", []byte("hello"), queue.WithQueue("critical"), queue.WithDelay(20*time.Millisecond))
	require.NoError(t, err)

	_, err = receiveCriticalWithTimeout(ctx, q, 20*time.Millisecond)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	var delivery queue.Delivery
	require.Eventually(t, func() bool {
		got, err := receiveCriticalWithTimeout(ctx, q, 50*time.Millisecond)
		if errors.Is(err, context.DeadlineExceeded) {
			return false
		}
		require.NoError(t, err)
		delivery = got
		return true
	}, time.Second, 10*time.Millisecond)
	require.NotNil(t, delivery)
	require.NotNil(t, delivery.Task())
	assert.Equal(t, "send_email", delivery.Task().Type)
}

func TestQueue_ReceiveAcksMalformedStreamMessage(t *testing.T) {
	t.Parallel()

	q, client := newRedisTestQueue(t)
	ctx := t.Context()
	require.NoError(t, q.ensureGroup(ctx, "critical"))
	require.NoError(t, client.XAdd(ctx, &goredis.XAddArgs{
		Stream: q.streamKey("critical"),
		Values: map[string]any{
			taskField: "{",
		},
	}).Err())

	delivery, err := receive(ctx, q, "critical")

	require.Error(t, err)
	assert.Nil(t, delivery)
	pending, err := client.XPending(ctx, q.streamKey("critical"), q.group).Result()
	require.NoError(t, err)
	assert.Zero(t, pending.Count)
}

func receive(ctx context.Context, q *Queue, queueName string) (queue.Delivery, error) {
	consumer, err := q.NewConsumer(ctx, queue.ConsumerConfig{Queue: queueName})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = consumer.Close()
	}()
	return consumer.Receive(ctx)
}

func receiveCriticalWithTimeout(ctx context.Context, q *Queue, timeout time.Duration) (queue.Delivery, error) {
	receiveCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return receive(receiveCtx, q, "critical")
}

func newRedisTestQueue(t *testing.T, opts ...Option) (*Queue, *goredis.Client) {
	t.Helper()

	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	client := goredis.NewClient(&goredis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		require.NoError(t, client.Close())
		t.Skipf("redis is not available at %s: %v", addr, err)
	}

	prefix := "queue-test:" + sanitizeRedisKey(t.Name()) + ":" + strconv.FormatInt(time.Now().UnixNano(), 10)
	options := append([]Option{
		WithPrefix(prefix),
		WithGroup("workers"),
		WithConsumer("worker-1"),
		WithPromoteSize(10),
	}, opts...)
	q := NewQueue(client, options...)

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(t.Context()), 2*time.Second)
		defer cleanupCancel()
		_ = client.Del(
			cleanupCtx,
			q.streamKey(queue.DefaultQueue),
			q.delayedKey(queue.DefaultQueue),
			q.deadLetterKey(queue.DefaultQueue),
			q.streamKey("critical"),
			q.delayedKey("critical"),
			q.deadLetterKey("critical"),
		).Err()
		_ = client.Close()
	})

	return q, client
}

func sanitizeRedisKey(name string) string {
	replacer := strings.NewReplacer("/", "-", " ", "-", ":", "-")
	return replacer.Replace(name)
}

func taskFromMessage(t *testing.T, message goredis.XMessage) queue.Task {
	t.Helper()

	value, ok := message.Values[taskField]
	require.True(t, ok)

	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		require.Failf(t, "unsupported task field", "%T", value)
	}

	var task queue.Task
	require.NoError(t, json.Unmarshal(data, &task))
	return task
}
