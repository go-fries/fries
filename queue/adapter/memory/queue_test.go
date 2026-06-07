package memory

import (
	"testing"
	"time"

	"github.com/go-fries/fries/queue/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueueHonorsDelay(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	q := NewQueue()
	_, err := queue.NewProducer(q).Enqueue(ctx, "delayed", nil, queue.WithDelay(20*time.Millisecond))
	require.NoError(t, err)

	_, err = q.Dequeue(ctx, queue.DefaultQueue, time.Minute)
	require.ErrorIs(t, err, queue.ErrNoTask)

	time.Sleep(30 * time.Millisecond)
	lease, err := q.Dequeue(ctx, queue.DefaultQueue, time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lease)
	require.NotNil(t, lease.Task())
	assert.Equal(t, "delayed", lease.Task().Type)
}

func TestQueueDeadLettersClonesTask(t *testing.T) {
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
