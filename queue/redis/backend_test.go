package redis

import (
	"encoding/json"
	"testing"

	"github.com/go-fries/fries/queue/v3"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLeaseFromMessage(t *testing.T) {
	t.Parallel()

	backend := NewBackend(nil)
	task := &queue.Task{
		ID:      "task-1",
		Type:    "send_email",
		Queue:   "default",
		Payload: []byte("hello"),
		Attempt: 2,
	}
	data, err := json.Marshal(task)
	require.NoError(t, err)

	lease, err := backend.leaseFromMessage(goredis.XMessage{
		ID: "1-0",
		Values: map[string]any{
			taskField: string(data),
		},
	})
	require.NoError(t, err)
	require.NotNil(t, lease)
	require.NotNil(t, lease.Task)

	assert.Equal(t, "1-0", lease.Token)
	assert.Equal(t, 3, lease.Task.Attempt)
	assert.Equal(t, "hello", string(lease.Task.Payload))
}

func TestBackendKeysUsePrefix(t *testing.T) {
	t.Parallel()

	backend := NewBackend(nil, WithPrefix("app:"), WithGroup("workers"), WithConsumer("worker-1"), WithPromoteSize(10))

	assert.Equal(t, "app:critical:stream", backend.streamKey("critical"))
	assert.Equal(t, "app:critical:delayed", backend.delayedKey("critical"))
	assert.Equal(t, "app:critical:dead", backend.deadLetterKey("critical"))
	assert.Equal(t, "workers", backend.group)
	assert.Equal(t, "worker-1", backend.consumer)
	assert.Equal(t, 10, backend.promoteSize)
}
