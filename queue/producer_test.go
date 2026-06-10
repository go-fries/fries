package queue

import (
	"context"
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

	delivery, err := q.Receive(ctx, DefaultQueue)
	require.NoError(t, err)
	require.NotNil(t, delivery)
	require.NotNil(t, delivery.Task())

	assert.Equal(t, "task-1", task.ID)
	assert.Equal(t, "hello", string(delivery.Task().Payload))
	assert.Equal(t, "1", delivery.Task().Metadata["trace"])
}

func TestProducer_RejectsEmptyTaskType(t *testing.T) {
	t.Parallel()

	_, err := NewProducer(newTestQueue()).Enqueue(t.Context(), "", nil)
	require.ErrorIs(t, err, ErrInvalidTaskType)
}

func TestEnqueueForWithCodec_RejectsNilProducer(t *testing.T) {
	t.Parallel()

	_, err := EnqueueForWithCodec(t.Context(), nil, "send_email", struct{}{}, nil)
	require.ErrorContains(t, err, "producer is nil")
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

	receiveCtx, cancel := context.WithTimeout(ctx, time.Millisecond)
	defer cancel()
	_, err = q.Receive(receiveCtx, DefaultQueue)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestProducer_WithQueueSetsDefaultQueue(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	q := newTestQueue()
	producer := NewProducer(q, WithQueue("critical"))

	task, err := producer.Enqueue(ctx, "send_email", nil)
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, "critical", task.Queue)

	delivery, err := q.Receive(ctx, "critical")
	require.NoError(t, err)
	require.NotNil(t, delivery)
	assert.Equal(t, "critical", delivery.Task().Queue)
}

func TestProducer_EnqueueQueueOverridesProducerQueue(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	q := newTestQueue()
	producer := NewProducer(q, WithQueue("critical"))

	task, err := producer.Enqueue(ctx, "send_email", nil, WithQueue("bulk"))
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, "bulk", task.Queue)

	delivery, err := q.Receive(ctx, "bulk")
	require.NoError(t, err)
	require.NotNil(t, delivery)
	assert.Equal(t, "bulk", delivery.Task().Queue)
}

func TestProducer_OptionsIgnoreEmptyOrInvalidValues(t *testing.T) {
	t.Parallel()

	task, err := NewProducer(newTestQueue(), WithQueue("")).Enqueue(
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
