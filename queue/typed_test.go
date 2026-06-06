package queue

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

type emailPayload struct {
	UserID  int    `json:"user_id"`
	Subject string `json:"subject"`
}

func TestEnqueueForEncodesPayload(t *testing.T) {
	t.Parallel()

	backend := NewMemoryBackend()
	task, err := EnqueueFor(t.Context(), NewProducer(backend), "send_email", emailPayload{
		UserID:  10,
		Subject: "welcome",
	})
	if err != nil {
		t.Fatalf("enqueue typed task: %v", err)
	}

	var decoded emailPayload
	if err := json.Unmarshal(task.Payload, &decoded); err != nil {
		t.Fatalf("unmarshal task payload: %v", err)
	}
	if decoded.UserID != 10 {
		t.Fatalf("user id = %d, want 10", decoded.UserID)
	}
	if decoded.Subject != "welcome" {
		t.Fatalf("subject = %q, want welcome", decoded.Subject)
	}
}

func TestHandleForDecodesPayload(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	backend := NewMemoryBackend()
	handled := make(chan *TaskFor[emailPayload], 1)
	worker := NewWorker(
		backend,
		HandleFor("send_email", HandlerFuncFor[emailPayload](func(_ context.Context, task *TaskFor[emailPayload]) error {
			handled <- task
			return nil
		})),
		WithPollInterval(time.Millisecond),
	)

	errs := make(chan error, 1)
	go func() {
		errs <- worker.Run(ctx)
	}()

	_, err := EnqueueFor(t.Context(), NewProducer(backend), "send_email", emailPayload{
		UserID:  12,
		Subject: "reset",
	})
	if err != nil {
		t.Fatalf("enqueue typed task: %v", err)
	}

	select {
	case task := <-handled:
		if task.ID == "" {
			t.Fatal("task id is empty")
		}
		if task.Payload.UserID != 12 {
			t.Fatalf("user id = %d, want 12", task.Payload.UserID)
		}
		if string(task.Task.Payload) == "" {
			t.Fatal("raw payload is empty")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for typed task")
	}

	cancel()
	if err := <-errs; err != nil {
		t.Fatalf("worker run: %v", err)
	}
}

type passthroughCodec struct{}

func (passthroughCodec) Marshal(data any) ([]byte, error) {
	return []byte(data.(string)), nil
}

func (passthroughCodec) Unmarshal(src []byte, dest any) error {
	*dest.(*string) = string(src)
	return nil
}

func TestHandleForWithCodecUsesCustomCodec(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	backend := NewMemoryBackend()
	handled := make(chan string, 1)
	worker := NewWorker(
		backend,
		HandleForWithCodec("raw", passthroughCodec{}, HandlerFuncFor[string](func(_ context.Context, task *TaskFor[string]) error {
			handled <- task.Payload
			return nil
		})),
		WithPollInterval(time.Millisecond),
	)

	errs := make(chan error, 1)
	go func() {
		errs <- worker.Run(ctx)
	}()

	_, err := EnqueueForWithCodec(t.Context(), NewProducer(backend), "raw", "payload", passthroughCodec{})
	if err != nil {
		t.Fatalf("enqueue typed task: %v", err)
	}

	select {
	case payload := <-handled:
		if payload != "payload" {
			t.Fatalf("payload = %q, want payload", payload)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for typed task")
	}

	cancel()
	if err := <-errs; err != nil {
		t.Fatalf("worker run: %v", err)
	}
}
