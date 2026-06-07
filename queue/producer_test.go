package queue

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProducerEnqueueCopiesTaskData(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	q := NewMemoryQueue()
	producer := NewProducer(q)

	payload := []byte("hello")
	metadata := map[string]string{"trace": "1"}

	task, err := producer.Enqueue(
		ctx, "send_email", payload,
		WithID("task-1"),
		WithMetadata(metadata),
		WithIdempotencyKey("email:1"),
	)
	require.NoError(t, err)

	payload[0] = 'x'
	metadata["trace"] = "2"

	lease, err := q.Dequeue(ctx, DefaultQueue, time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lease)
	require.NotNil(t, lease.Task)

	assert.Equal(t, "task-1", task.ID)
	assert.Equal(t, "hello", string(lease.Task.Payload))
	assert.Equal(t, "1", lease.Task.Metadata["trace"])
	assert.Equal(t, "email:1", lease.Task.IdempotencyKey)
}

func TestProducerRejectsEmptyTaskType(t *testing.T) {
	t.Parallel()

	_, err := NewProducer(NewMemoryQueue()).Enqueue(t.Context(), "", nil)
	require.ErrorIs(t, err, ErrInvalidTaskType)
}

func TestMemoryQueueHonorsDelay(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	q := NewMemoryQueue()
	_, err := NewProducer(q).Enqueue(ctx, "delayed", nil, WithDelay(20*time.Millisecond))
	require.NoError(t, err)

	_, err = q.Dequeue(ctx, DefaultQueue, time.Minute)
	require.ErrorIs(t, err, ErrNoTask)

	time.Sleep(30 * time.Millisecond)
	lease, err := q.Dequeue(ctx, DefaultQueue, time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lease)
	require.NotNil(t, lease.Task)
	assert.Equal(t, "delayed", lease.Task.Type)
}
