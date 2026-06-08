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

	"github.com/go-fries/fries/queue/v3"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueue_LeaseFromMessage(t *testing.T) {
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

	lease, err := q.leaseFromMessage(goredis.XMessage{
		ID: "1-0",
		Values: map[string]any{
			taskField: string(data),
		},
	})
	require.NoError(t, err)
	require.NotNil(t, lease)
	require.NotNil(t, lease.Task())

	l, ok := lease.(*redisLease)
	require.True(t, ok)
	assert.Equal(t, "1-0", l.streamID)
	assert.Equal(t, 3, lease.Task().Attempt)
	assert.Equal(t, "hello", string(lease.Task().Payload))
}

func TestQueue_LeaseFromMessageAcceptsBytes(t *testing.T) {
	t.Parallel()

	q := NewQueue(nil)
	data, err := json.Marshal(&queue.Task{ID: "task-1", Type: "send_email"})
	require.NoError(t, err)

	lease, err := q.leaseFromMessage(goredis.XMessage{
		ID: "1-0",
		Values: map[string]any{
			taskField: data,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, lease)
	require.NotNil(t, lease.Task())
	assert.Equal(t, "task-1", lease.Task().ID)
	assert.Equal(t, 1, lease.Task().Attempt)
}

func TestQueue_LeaseFromMessageWithDeliveryCount(t *testing.T) {
	t.Parallel()

	q := NewQueue(nil)
	data, err := json.Marshal(&queue.Task{ID: "task-1", Type: "send_email", Attempt: 1})
	require.NoError(t, err)

	lease, err := q.leaseFromMessageWithDeliveryCount(goredis.XMessage{
		ID: "1-0",
		Values: map[string]any{
			taskField: data,
		},
	}, 2)
	require.NoError(t, err)
	require.NotNil(t, lease)
	require.NotNil(t, lease.Task())
	assert.Equal(t, 3, lease.Task().Attempt)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := attemptWithDeliveryCount(tt.baseAttempt, tt.deliveryCount)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestQueue_LeaseFromMessageErrors(t *testing.T) {
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

	q := NewQueue(nil, WithPrefix(""), WithGroup(""), WithConsumer(""), WithPromoteSize(0))

	assert.Equal(t, "queue:critical:stream", q.streamKey("critical"))
	assert.Equal(t, "queue:critical:delayed", q.delayedKey("critical"))
	assert.Equal(t, "queue:critical:dead", q.deadLetterKey("critical"))
	assert.Equal(t, defaultGroup, q.group)
	assert.Equal(t, defaultConsumer, q.consumer)
	assert.Equal(t, defaultPromoteBy, q.promoteSize)
}

func TestQueue_Options(t *testing.T) {
	t.Parallel()

	q := NewQueue(nil, WithPrefix("app:"), WithGroup("workers"), WithConsumer("worker-1"), WithPromoteSize(10))

	assert.Equal(t, "app:critical:stream", q.streamKey("critical"))
	assert.Equal(t, "app:critical:delayed", q.delayedKey("critical"))
	assert.Equal(t, "app:critical:dead", q.deadLetterKey("critical"))
	assert.Equal(t, "workers", q.group)
	assert.Equal(t, "worker-1", q.consumer)
	assert.Equal(t, 10, q.promoteSize)
}

func TestQueue_NoopOperations(t *testing.T) {
	t.Parallel()

	q := NewQueue(nil)

	require.NoError(t, q.Enqueue(t.Context(), nil))
	require.NoError(t, q.Ack(t.Context(), nil))
	require.NoError(t, q.Ack(t.Context(), queue.NewLease(&queue.Task{})))
	require.NoError(t, q.Retry(t.Context(), nil, 0))
	require.NoError(t, q.Retry(t.Context(), queue.NewLease(nil), 0))
	require.NoError(t, q.DeadLetter(t.Context(), nil, "failed"))
	require.NoError(t, q.DeadLetter(t.Context(), queue.NewLease(nil), "failed"))
}

func TestRedisLease_TaskNilReceiver(t *testing.T) {
	t.Parallel()

	var lease *redisLease

	assert.Nil(t, lease.Task())
}

func TestQueue_EnqueueDequeueAck(t *testing.T) {
	t.Parallel()

	q, _ := newRedisTestQueue(t)
	ctx := t.Context()

	_, err := queue.NewProducer(q).Enqueue(ctx, "send_email", []byte("hello"), queue.WithQueue("critical"))
	require.NoError(t, err)

	lease, err := q.Dequeue(ctx, "critical", time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lease)
	require.NotNil(t, lease.Task())
	assert.Equal(t, "send_email", lease.Task().Type)
	assert.Equal(t, "critical", lease.Task().Queue)
	assert.Equal(t, []byte("hello"), lease.Task().Payload)
	assert.Equal(t, 1, lease.Task().Attempt)

	require.NoError(t, q.Ack(ctx, lease))

	_, err = q.Dequeue(ctx, "critical", time.Minute)
	require.ErrorIs(t, err, queue.ErrNoTask)
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

	lease, err := q.Dequeue(ctx, queue.DefaultQueue, time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lease)
	require.NotNil(t, lease.Task())
	assert.Equal(t, queue.DefaultQueue, lease.Task().Queue)
}

func TestQueue_RetryReenqueuesAndAcksLease(t *testing.T) {
	t.Parallel()

	q, _ := newRedisTestQueue(t)
	ctx := t.Context()

	_, err := queue.NewProducer(q).Enqueue(ctx, "send_email", []byte("hello"), queue.WithQueue("critical"))
	require.NoError(t, err)

	lease, err := q.Dequeue(ctx, "critical", time.Minute)
	require.NoError(t, err)
	require.NoError(t, q.Retry(ctx, lease, 0))

	lease, err = q.Dequeue(ctx, "critical", time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lease)
	require.NotNil(t, lease.Task())
	assert.Equal(t, "send_email", lease.Task().Type)
	assert.Equal(t, 2, lease.Task().Attempt)
}

func TestQueue_ClaimPendingIncrementsAttempt(t *testing.T) {
	t.Parallel()

	q, client := newRedisTestQueue(t)
	ctx := t.Context()

	_, err := queue.NewProducer(q).Enqueue(ctx, "send_email", []byte("hello"), queue.WithQueue("critical"))
	require.NoError(t, err)

	lease, err := q.Dequeue(ctx, "critical", time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lease)
	require.NotNil(t, lease.Task())
	assert.Equal(t, 1, lease.Task().Attempt)

	messages, err := client.XRange(ctx, q.streamKey("critical"), "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, messages, 1)
	stored := taskFromMessage(t, messages[0])
	assert.Equal(t, 0, stored.Attempt)

	var claimed queue.Lease
	require.Eventually(t, func() bool {
		got, err := q.Dequeue(ctx, "critical", time.Millisecond)
		if errors.Is(err, queue.ErrNoTask) {
			return false
		}
		require.NoError(t, err)
		claimed = got
		return true
	}, time.Second, 10*time.Millisecond)

	require.NotNil(t, claimed)
	require.NotNil(t, claimed.Task())
	assert.Equal(t, "send_email", claimed.Task().Type)
	assert.Equal(t, 2, claimed.Task().Attempt)
	require.NoError(t, q.Ack(ctx, claimed))
}

func TestQueue_ClaimRetriedPendingIncrementsAttempt(t *testing.T) {
	t.Parallel()

	q, client := newRedisTestQueue(t)
	ctx := t.Context()

	_, err := queue.NewProducer(q).Enqueue(ctx, "send_email", []byte("hello"), queue.WithQueue("critical"))
	require.NoError(t, err)

	lease, err := q.Dequeue(ctx, "critical", time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lease)
	require.NotNil(t, lease.Task())
	require.Equal(t, 1, lease.Task().Attempt)
	require.NoError(t, q.Retry(ctx, lease, 0))

	lease, err = q.Dequeue(ctx, "critical", time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lease)
	require.NotNil(t, lease.Task())
	assert.Equal(t, 2, lease.Task().Attempt)

	messages, err := client.XRange(ctx, q.streamKey("critical"), "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, messages, 2)
	stored := taskFromMessage(t, messages[1])
	assert.Equal(t, 1, stored.Attempt)

	var claimed queue.Lease
	require.Eventually(t, func() bool {
		got, err := q.Dequeue(ctx, "critical", time.Millisecond)
		if errors.Is(err, queue.ErrNoTask) {
			return false
		}
		require.NoError(t, err)
		claimed = got
		return true
	}, time.Second, 10*time.Millisecond)

	require.NotNil(t, claimed)
	require.NotNil(t, claimed.Task())
	assert.Equal(t, "send_email", claimed.Task().Type)
	assert.Equal(t, 3, claimed.Task().Attempt)
	require.NoError(t, q.Ack(ctx, claimed))
}

func TestQueue_DeadLetterWritesReasonAndAcksLease(t *testing.T) {
	t.Parallel()

	q, client := newRedisTestQueue(t)
	ctx := t.Context()

	_, err := queue.NewProducer(q).Enqueue(ctx, "send_email", []byte("hello"), queue.WithQueue("critical"))
	require.NoError(t, err)

	lease, err := q.Dequeue(ctx, "critical", time.Minute)
	require.NoError(t, err)
	require.NoError(t, q.DeadLetter(ctx, lease, "failed"))

	messages, err := client.XRange(ctx, q.deadLetterKey("critical"), "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, messages, 1)
	assert.Equal(t, "failed", messages[0].Values[deadReasonField])

	_, err = q.Dequeue(ctx, "critical", time.Minute)
	require.ErrorIs(t, err, queue.ErrNoTask)
}

func TestQueue_DelayedTaskPromotion(t *testing.T) {
	t.Parallel()

	q, _ := newRedisTestQueue(t)
	ctx := t.Context()

	_, err := queue.NewProducer(q).Enqueue(ctx, "send_email", []byte("hello"), queue.WithQueue("critical"), queue.WithDelay(20*time.Millisecond))
	require.NoError(t, err)

	_, err = q.Dequeue(ctx, "critical", time.Minute)
	require.ErrorIs(t, err, queue.ErrNoTask)

	var lease queue.Lease
	require.Eventually(t, func() bool {
		got, err := q.Dequeue(ctx, "critical", time.Minute)
		if errors.Is(err, queue.ErrNoTask) {
			return false
		}
		require.NoError(t, err)
		lease = got
		return true
	}, time.Second, 10*time.Millisecond)
	require.NotNil(t, lease)
	require.NotNil(t, lease.Task())
	assert.Equal(t, "send_email", lease.Task().Type)
}

func newRedisTestQueue(t *testing.T) (*Queue, *goredis.Client) {
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
	q := NewQueue(client, WithPrefix(prefix), WithGroup("workers"), WithConsumer("worker-1"), WithPromoteSize(10))

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
