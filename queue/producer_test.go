package queue

import (
	"errors"
	"testing"
	"time"
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
	if err != nil {
		t.Fatalf("enqueue task: %v", err)
	}

	payload[0] = 'x'
	headers["trace"] = "2"

	lease, err := backend.Dequeue(ctx, DefaultQueue, time.Minute)
	if err != nil {
		t.Fatalf("dequeue task: %v", err)
	}

	if task.ID != "task-1" {
		t.Fatalf("task id = %q, want task-1", task.ID)
	}
	if string(lease.Task.Payload) != "hello" {
		t.Fatalf("payload = %q, want hello", lease.Task.Payload)
	}
	if lease.Task.Headers["trace"] != "1" {
		t.Fatalf("header trace = %q, want 1", lease.Task.Headers["trace"])
	}
	if lease.Task.IdempotencyKey != "email:1" {
		t.Fatalf("idempotency key = %q, want email:1", lease.Task.IdempotencyKey)
	}
}

func TestProducerRejectsEmptyTaskType(t *testing.T) {
	t.Parallel()

	_, err := NewProducer(NewMemoryBackend()).Enqueue(t.Context(), "", nil)
	if !errors.Is(err, ErrInvalidTaskType) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidTaskType)
	}
}

func TestMemoryBackendHonorsDelay(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	backend := NewMemoryBackend()
	_, err := NewProducer(backend).Enqueue(ctx, "delayed", nil, WithDelay(20*time.Millisecond))
	if err != nil {
		t.Fatalf("enqueue task: %v", err)
	}

	if _, err := backend.Dequeue(ctx, DefaultQueue, time.Minute); !errors.Is(err, ErrNoTask) {
		t.Fatalf("immediate dequeue error = %v, want %v", err, ErrNoTask)
	}

	time.Sleep(30 * time.Millisecond)
	lease, err := backend.Dequeue(ctx, DefaultQueue, time.Minute)
	if err != nil {
		t.Fatalf("delayed dequeue: %v", err)
	}
	if lease.Task.Type != "delayed" {
		t.Fatalf("task type = %q, want delayed", lease.Task.Type)
	}
}
