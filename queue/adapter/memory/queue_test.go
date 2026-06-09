package memory

import (
	"context"
	"testing"
	"time"

	"github.com/go-fries/fries/queue/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueue_EnqueueDefaultsQueueAndClonesTask(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	q := NewQueue()
	task := &queue.Task{
		ID:       "task-1",
		Type:     "send_email",
		Payload:  []byte("hello"),
		Metadata: map[string]string{"trace": "1"},
	}

	require.NoError(t, q.Enqueue(ctx, task))
	task.Payload[0] = 'x'
	task.Metadata["trace"] = "2"

	delivery, err := receive(ctx, q, "")
	require.NoError(t, err)
	require.NotNil(t, delivery)
	require.NotNil(t, delivery.Task())

	assert.Equal(t, queue.DefaultQueue, delivery.Task().Queue)
	assert.Equal(t, 1, delivery.Task().Attempt)
	assert.Equal(t, "hello", string(delivery.Task().Payload))
	assert.Equal(t, "1", delivery.Task().Metadata["trace"])
}

func TestQueue_ReceiveHonorsAvailability(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	q := NewQueue()
	now := time.Now().UTC()
	require.NoError(t, q.Enqueue(ctx, &queue.Task{
		ID:          "future",
		Type:        "send_email",
		AvailableAt: now.Add(time.Minute),
	}))

	receiveCtx, cancel := context.WithTimeout(ctx, time.Millisecond)
	_, err := receive(receiveCtx, q, queue.DefaultQueue)
	cancel()
	require.ErrorIs(t, err, context.DeadlineExceeded)

	require.NoError(t, q.Enqueue(ctx, &queue.Task{
		ID:          "ready",
		Type:        "send_email",
		AvailableAt: now.Add(-time.Minute),
	}))

	delivery, err := receive(ctx, q, queue.DefaultQueue)
	require.NoError(t, err)
	require.NotNil(t, delivery)
	require.NotNil(t, delivery.Task())
	assert.Equal(t, "ready", delivery.Task().ID)
	assert.Equal(t, 1, delivery.Task().Attempt)
}

func TestQueue_RetryReenqueuesTask(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	q := NewQueue()
	require.NoError(t, q.Enqueue(ctx, &queue.Task{
		ID:    "task-1",
		Type:  "send_email",
		Queue: "critical",
	}))

	delivery, err := receive(ctx, q, "critical")
	require.NoError(t, err)
	require.NoError(t, delivery.Retry(ctx, 0))

	delivery, err = receive(ctx, q, "critical")
	require.NoError(t, err)
	require.NotNil(t, delivery)
	require.NotNil(t, delivery.Task())
	assert.Equal(t, "task-1", delivery.Task().ID)
	assert.Equal(t, 2, delivery.Task().Attempt)
}

func TestQueue_DeadLettersClonesTask(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	q := NewQueue()
	task := &queue.Task{
		ID:       "task-1",
		Type:     "send_email",
		Queue:    "critical",
		Payload:  []byte("hello"),
		Metadata: map[string]string{"trace": "1"},
	}

	require.NoError(t, q.Enqueue(ctx, task))
	task.Payload[0] = 'x'
	task.Metadata["trace"] = "2"

	delivery, err := receive(ctx, q, "critical")
	require.NoError(t, err)
	require.NoError(t, delivery.DeadLetter(ctx, "failed"))

	dead := q.DeadLetters("critical")
	require.Len(t, dead, 1)
	assert.Equal(t, "hello", string(dead[0].Payload))
	assert.Equal(t, "1", dead[0].Metadata["trace"])
	assert.Equal(t, "failed", dead[0].Metadata["queue.dead_letter.reason"])

	dead[0].Payload[0] = 'z'
	dead[0].Metadata["trace"] = "3"

	dead = q.DeadLetters("critical")
	require.Len(t, dead, 1)
	assert.Equal(t, "hello", string(dead[0].Payload))
	assert.Equal(t, "1", dead[0].Metadata["trace"])
}

func TestQueue_MethodsReturnContextError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	q := NewQueue()
	task := &queue.Task{ID: "task-1", Type: "send_email"}

	require.ErrorIs(t, q.Enqueue(ctx, task), context.Canceled)
	consumer, err := q.NewConsumer(t.Context(), queue.ConsumerConfig{Queue: queue.DefaultQueue})
	require.NoError(t, err)
	_, err = consumer.Receive(ctx)
	require.ErrorIs(t, err, context.Canceled)

	delivery := &delivery{queue: q, task: task}
	require.ErrorIs(t, delivery.Ack(ctx), context.Canceled)
	require.ErrorIs(t, delivery.Retry(ctx, 0), context.Canceled)
	require.ErrorIs(t, delivery.DeadLetter(ctx, "failed"), context.Canceled)
}

func TestQueue_ReceiveReturnsConsumerClosed(t *testing.T) {
	t.Parallel()

	q := NewQueue()
	consumer, err := q.NewConsumer(t.Context(), queue.ConsumerConfig{})
	require.NoError(t, err)
	require.NoError(t, consumer.Close())

	delivery, err := consumer.Receive(t.Context())

	require.ErrorIs(t, err, queue.ErrConsumerClosed)
	assert.Nil(t, delivery)
}

func TestQueue_NilDeliveryOperationsAreNoop(t *testing.T) {
	t.Parallel()

	q := NewQueue()
	var nilDelivery *delivery

	require.NoError(t, q.Enqueue(t.Context(), nil))
	require.NoError(t, nilDelivery.Retry(t.Context(), 0))
	require.NoError(t, nilDelivery.DeadLetter(t.Context(), "failed"))
	require.NoError(t, (&delivery{queue: q}).Retry(t.Context(), 0))
	require.NoError(t, (&delivery{queue: q}).DeadLetter(t.Context(), "failed"))
	assert.Empty(t, q.DeadLetters(queue.DefaultQueue))
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
