package queue

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTask_Clone(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 6, 7, 10, 0, 0, 0, time.UTC)
	availableAt := createdAt.Add(time.Minute)
	task := &Task{
		ID:          "task-1",
		Type:        "send_email",
		Queue:       "critical",
		Payload:     []byte("hello"),
		Metadata:    map[string]string{"trace": "1"},
		Attempt:     2,
		CreatedAt:   createdAt,
		AvailableAt: availableAt,
	}

	cloned := task.Clone()
	require.NotNil(t, cloned)
	require.NotSame(t, task, cloned)

	assert.Equal(t, task.ID, cloned.ID)
	assert.Equal(t, task.Type, cloned.Type)
	assert.Equal(t, task.Queue, cloned.Queue)
	assert.Equal(t, task.Attempt, cloned.Attempt)
	assert.Equal(t, task.CreatedAt, cloned.CreatedAt)
	assert.Equal(t, task.AvailableAt, cloned.AvailableAt)
	assert.Equal(t, []byte("hello"), cloned.Payload)
	assert.Equal(t, map[string]string{"trace": "1"}, cloned.Metadata)

	task.Payload[0] = 'x'
	task.Metadata["trace"] = "2"

	assert.Equal(t, []byte("hello"), cloned.Payload)
	assert.Equal(t, "1", cloned.Metadata["trace"])
}

func TestTask_CloneNil(t *testing.T) {
	t.Parallel()

	var task *Task

	assert.Nil(t, task.Clone())
}

func TestDelivery_Task(t *testing.T) {
	t.Parallel()

	task := &Task{ID: "task-1"}
	delivery := NewDelivery(task)

	require.NotNil(t, delivery)
	assert.Same(t, task, delivery.Task())
}

func TestDelivery_TaskNilReceiver(t *testing.T) {
	t.Parallel()

	var delivery *noopDelivery

	assert.Nil(t, delivery.Task())
}
