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
	backend := NewMemoryBackend()
	producer := NewProducer(backend)

	payload := []byte("hello")
	headers := map[string]string{"trace": "1"}

	task, err := producer.Enqueue(
		ctx, "send_email", payload,
		WithID("task-1"),
		WithHeaders(headers),
		WithIdempotencyKey("email:1"),
	)
	require.NoError(t, err)

	payload[0] = 'x'
	headers["trace"] = "2"

	lease, err := backend.Dequeue(ctx, DefaultQueue, time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lease)
	require.NotNil(t, lease.Task)

	assert.Equal(t, "task-1", task.ID)
	assert.Equal(t, "hello", string(lease.Task.Payload))
	assert.Equal(t, "1", lease.Task.Headers["trace"])
	assert.Equal(t, "email:1", lease.Task.IdempotencyKey)
}

func TestProducerRejectsEmptyTaskType(t *testing.T) {
	t.Parallel()

	_, err := NewProducer(NewMemoryBackend()).Enqueue(t.Context(), "", nil)
	require.ErrorIs(t, err, ErrInvalidTaskType)
}

func TestMemoryBackendHonorsDelay(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	backend := NewMemoryBackend()
	_, err := NewProducer(backend).Enqueue(ctx, "delayed", nil, WithDelay(20*time.Millisecond))
	require.NoError(t, err)

	_, err = backend.Dequeue(ctx, DefaultQueue, time.Minute)
	require.ErrorIs(t, err, ErrNoTask)

	time.Sleep(30 * time.Millisecond)
	lease, err := backend.Dequeue(ctx, DefaultQueue, time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lease)
	require.NotNil(t, lease.Task)
	assert.Equal(t, "delayed", lease.Task.Type)
}
