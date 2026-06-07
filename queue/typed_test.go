package queue

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type emailPayload struct {
	UserID  int    `json:"user_id"`
	Subject string `json:"subject"`
}

func TestEnqueueForEncodesPayload(t *testing.T) {
	t.Parallel()

	q := newTestQueue()
	task, err := EnqueueFor(t.Context(), NewProducer(q), "send_email", emailPayload{
		UserID:  10,
		Subject: "welcome",
	})
	require.NoError(t, err)
	require.NotNil(t, task)

	var decoded emailPayload
	require.NoError(t, json.Unmarshal(task.Payload, &decoded))
	assert.Equal(t, 10, decoded.UserID)
	assert.Equal(t, "welcome", decoded.Subject)
}

func TestHandleForDecodesPayload(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	q := newTestQueue()
	handled := make(chan *TaskFor[emailPayload], 1)
	worker := NewWorker(
		q,
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

	_, err := EnqueueFor(t.Context(), NewProducer(q), "send_email", emailPayload{
		UserID:  12,
		Subject: "reset",
	})
	require.NoError(t, err)

	select {
	case task := <-handled:
		require.NotNil(t, task.Task)
		assert.NotEmpty(t, task.Task.ID)
		assert.Equal(t, 12, task.Payload.UserID)
		assert.NotEmpty(t, string(task.Task.Payload))
	case <-time.After(time.Second):
		require.Fail(t, "timeout waiting for typed task")
	}

	cancel()
	require.NoError(t, <-errs)
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

	q := newTestQueue()
	handled := make(chan string, 1)
	worker := NewWorker(
		q,
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

	_, err := EnqueueForWithCodec(t.Context(), NewProducer(q), "raw", "payload", passthroughCodec{})
	require.NoError(t, err)

	select {
	case payload := <-handled:
		assert.Equal(t, "payload", payload)
	case <-time.After(time.Second):
		require.Fail(t, "timeout waiting for typed task")
	}

	cancel()
	require.NoError(t, <-errs)
}
