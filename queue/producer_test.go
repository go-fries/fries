package queue

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProducer_EnqueueCopiesTaskData(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	q := newTestQueue()
	producer := NewProducer(q)

	payload := []byte("hello")
	metadata := map[string]string{"trace": "1"}

	task, err := producer.Enqueue(
		ctx, "send_email", payload,
		WithID("task-1"),
		WithMetadata(metadata),
	)
	require.NoError(t, err)

	payload[0] = 'x'
	metadata["trace"] = "2"

	lease, err := q.Dequeue(ctx, DefaultQueue, time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lease)
	require.NotNil(t, lease.Task())

	assert.Equal(t, "task-1", task.ID)
	assert.Equal(t, "hello", string(lease.Task().Payload))
	assert.Equal(t, "1", lease.Task().Metadata["trace"])
}

func TestProducer_RejectsEmptyTaskType(t *testing.T) {
	t.Parallel()

	_, err := NewProducer(newTestQueue()).Enqueue(t.Context(), "", nil)
	require.ErrorIs(t, err, ErrInvalidTaskType)
}

func TestProducer_EnqueueSetsDefaultsAndOptions(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	q := newTestQueue()
	before := time.Now().UTC()

	task, err := NewProducer(q).Enqueue(
		ctx,
		"delayed",
		nil,
		WithDelay(20*time.Millisecond),
		WithMetadataValue("trace", "1"),
		WithMetadata(map[string]string{"tenant": "acme"}),
	)
	require.NoError(t, err)
	require.NotNil(t, task)

	after := time.Now().UTC()

	assert.NotEmpty(t, task.ID)
	assert.Equal(t, DefaultQueue, task.Queue)
	assert.Equal(t, map[string]string{
		"tenant": "acme",
		"trace":  "1",
	}, task.Metadata)
	assert.False(t, task.CreatedAt.Before(before))
	assert.False(t, task.CreatedAt.After(after))
	assert.Equal(t, time.UTC, task.CreatedAt.Location())
	assert.Equal(t, time.UTC, task.AvailableAt.Location())
	assert.GreaterOrEqual(t, task.AvailableAt.Sub(task.CreatedAt), 20*time.Millisecond)

	_, err = q.Dequeue(ctx, DefaultQueue, time.Minute)
	require.ErrorIs(t, err, ErrNoTask)
}

func TestProducer_OptionsIgnoreEmptyOrInvalidValues(t *testing.T) {
	t.Parallel()

	task, err := NewProducer(newTestQueue()).Enqueue(
		t.Context(),
		"send_email",
		nil,
		WithID(""),
		WithQueue(""),
		WithDelay(-time.Second),
		WithMetadata(nil),
	)
	require.NoError(t, err)
	require.NotNil(t, task)

	assert.NotEmpty(t, task.ID)
	assert.Equal(t, DefaultQueue, task.Queue)
	assert.Nil(t, task.Metadata)
	assert.Equal(t, task.CreatedAt, task.AvailableAt)
}
