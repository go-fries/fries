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

	lease, err := q.Dequeue(ctx, "", time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lease)
	require.NotNil(t, lease.Task())

	assert.Equal(t, queue.DefaultQueue, lease.Task().Queue)
	assert.Equal(t, 1, lease.Task().Attempt)
	assert.Equal(t, "hello", string(lease.Task().Payload))
	assert.Equal(t, "1", lease.Task().Metadata["trace"])
}

func TestQueue_DequeueHonorsAvailability(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	q := NewQueue()
	now := time.Now().UTC()
	require.NoError(t, q.Enqueue(ctx, &queue.Task{
		ID:          "future",
		Type:        "send_email",
		AvailableAt: now.Add(time.Minute),
	}))

	_, err := q.Dequeue(ctx, queue.DefaultQueue, time.Minute)
	require.ErrorIs(t, err, queue.ErrNoTask)

	require.NoError(t, q.Enqueue(ctx, &queue.Task{
		ID:          "ready",
		Type:        "send_email",
		AvailableAt: now.Add(-time.Minute),
	}))

	lease, err := q.Dequeue(ctx, queue.DefaultQueue, time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lease)
	require.NotNil(t, lease.Task())
	assert.Equal(t, "ready", lease.Task().ID)
	assert.Equal(t, 1, lease.Task().Attempt)
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

	lease, err := q.Dequeue(ctx, "critical", time.Minute)
	require.NoError(t, err)
	require.NoError(t, q.Retry(ctx, lease, 0))

	lease, err = q.Dequeue(ctx, "critical", time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lease)
	require.NotNil(t, lease.Task())
	assert.Equal(t, "task-1", lease.Task().ID)
	assert.Equal(t, 2, lease.Task().Attempt)
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

	lease, err := q.Dequeue(ctx, "critical", time.Minute)
	require.NoError(t, err)
	require.NoError(t, q.DeadLetter(ctx, lease, "failed"))

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
	_, err := q.Dequeue(ctx, queue.DefaultQueue, time.Minute)
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorIs(t, q.Ack(ctx, nil), context.Canceled)
	require.ErrorIs(t, q.Retry(ctx, queue.NewLease(task), 0), context.Canceled)
	require.ErrorIs(t, q.DeadLetter(ctx, queue.NewLease(task), "failed"), context.Canceled)
}

func TestQueue_NilLeaseOperationsAreNoop(t *testing.T) {
	t.Parallel()

	q := NewQueue()

	require.NoError(t, q.Enqueue(t.Context(), nil))
	require.NoError(t, q.Retry(t.Context(), nil, 0))
	require.NoError(t, q.Retry(t.Context(), queue.NewLease(nil), 0))
	require.NoError(t, q.DeadLetter(t.Context(), nil, "failed"))
	require.NoError(t, q.DeadLetter(t.Context(), queue.NewLease(nil), "failed"))
	assert.Empty(t, q.DeadLetters(queue.DefaultQueue))
}
