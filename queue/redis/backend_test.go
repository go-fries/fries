package redis

import (
	"encoding/json"
	"testing"

	"github.com/go-fries/fries/queue/v3"
	goredis "github.com/redis/go-redis/v9"
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
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	lease, err := backend.leaseFromMessage(goredis.XMessage{
		ID: "1-0",
		Values: map[string]any{
			taskField: string(data),
		},
	})
	if err != nil {
		t.Fatalf("lease from message: %v", err)
	}

	if lease.Token != "1-0" {
		t.Fatalf("token = %q, want 1-0", lease.Token)
	}
	if lease.Task.Attempt != 3 {
		t.Fatalf("attempt = %d, want 3", lease.Task.Attempt)
	}
	if string(lease.Task.Payload) != "hello" {
		t.Fatalf("payload = %q, want hello", lease.Task.Payload)
	}
}

func TestBackendKeysUsePrefix(t *testing.T) {
	t.Parallel()

	backend := NewBackend(nil, WithPrefix("app:"), WithGroup("workers"), WithConsumer("worker-1"), WithPromoteSize(10))

	if backend.streamKey("critical") != "app:critical:stream" {
		t.Fatalf("stream key = %q", backend.streamKey("critical"))
	}
	if backend.delayedKey("critical") != "app:critical:delayed" {
		t.Fatalf("delayed key = %q", backend.delayedKey("critical"))
	}
	if backend.deadLetterKey("critical") != "app:critical:dead" {
		t.Fatalf("dead letter key = %q", backend.deadLetterKey("critical"))
	}
	if backend.group != "workers" {
		t.Fatalf("group = %q, want workers", backend.group)
	}
	if backend.consumer != "worker-1" {
		t.Fatalf("consumer = %q, want worker-1", backend.consumer)
	}
	if backend.promoteSize != 10 {
		t.Fatalf("promote size = %d, want 10", backend.promoteSize)
	}
}
