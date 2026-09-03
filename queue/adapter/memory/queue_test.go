package memory

import (
	"context"
	"testing"
	"time"

	"github.com/go-fries/fries/queue/v4"
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

func TestQueue_NewConsumerReturnsContextError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	consumer, err := NewQueue().NewConsumer(ctx, queue.ConsumerConfig{})

	assert.Nil(t, consumer)
	require.ErrorIs(t, err, context.Canceled)
}

func TestQueue_ReceiveWaitsUntilTaskIsAvailable(t *testing.T) {
	t.Parallel()

	q := NewQueue()
	require.NoError(t, q.Enqueue(t.Context(), &queue.Task{
		ID:          "delayed",
		Type:        "send_email",
		AvailableAt: time.Now().UTC().Add(50 * time.Millisecond),
	}))
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	delivery, err := receive(ctx, q, queue.DefaultQueue)

	require.NoError(t, err)
	require.NotNil(t, delivery)
	assert.Equal(t, "delayed", delivery.Task().ID)
}

func TestQueue_ReceiveWakesWhenTaskIsEnqueued(t *testing.T) {
	q := NewQueue()
	consumer, err := q.NewConsumer(t.Context(), queue.ConsumerConfig{})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, consumer.Close())
	})
	type result struct {
		delivery queue.Delivery
		err      error
	}
	receiving := make(chan struct{})
	received := make(chan result, 1)
	go func() {
		close(receiving)
		delivery, err := consumer.Receive(t.Context())
		received <- result{delivery: delivery, err: err}
	}()
	<-receiving

	select {
	case result := <-received:
		require.FailNowf(t, "receive returned before enqueue", "%v", result.err)
	case <-time.After(20 * time.Millisecond):
	}
	require.NoError(t, q.Enqueue(t.Context(), &queue.Task{
		ID:   "enqueued",
		Type: "send_email",
	}))

	select {
	case result := <-received:
		require.NoError(t, result.err)
		require.NotNil(t, result.delivery)
		assert.Equal(t, "enqueued", result.delivery.Task().ID)
	case <-time.After(time.Second):
		require.FailNow(t, "receive did not wake after enqueue")
	}
}

func TestQueue_NilDeliveryHasNoTask(t *testing.T) {
	t.Parallel()

	var delivery *delivery

	assert.Nil(t, delivery.Task())
}

func TestQueue_DeadLetterInitializesMetadataAndUsesDefaultQueue(t *testing.T) {
	t.Parallel()

	q := NewQueue()
	require.NoError(t, q.Enqueue(t.Context(), &queue.Task{
		ID:   "task-1",
		Type: "send_email",
	}))
	delivery, err := receive(t.Context(), q, queue.DefaultQueue)
	require.NoError(t, err)

	require.NoError(t, delivery.DeadLetter(t.Context(), "failed"))

	dead := q.DeadLetters("")
	require.Len(t, dead, 1)
	assert.Equal(t, queue.DefaultQueue, dead[0].Queue)
	assert.Equal(t, "failed", dead[0].Metadata["queue.dead_letter.reason"])
}

func TestStopTimerHandlesStoppedTimer(t *testing.T) {
	t.Parallel()

	timer := time.NewTimer(time.Hour)
	require.True(t, timer.Stop())

	stopTimer(timer)
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
