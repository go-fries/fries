package queue

import (
	"context"
	"encoding/json"
	"errors"
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

type failingCodec struct {
	marshalErr   error
	unmarshalErr error
}

func (c failingCodec) Marshal(any) ([]byte, error) {
	return nil, c.marshalErr
}

func (c failingCodec) Unmarshal([]byte, any) error {
	return c.unmarshalErr
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

func TestEnqueueForWithCodecUsesDefaultCodecWhenNil(t *testing.T) {
	t.Parallel()

	task, err := EnqueueForWithCodec(t.Context(), NewProducer(newTestQueue()), "send_email", emailPayload{
		UserID:  30,
		Subject: "welcome",
	}, nil)
	require.NoError(t, err)
	require.NotNil(t, task)

	var decoded emailPayload
	require.NoError(t, json.Unmarshal(task.Payload, &decoded))
	assert.Equal(t, 30, decoded.UserID)
	assert.Equal(t, "welcome", decoded.Subject)
}

func TestEnqueueForWithCodecReturnsMarshalError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("marshal failed")

	_, err := EnqueueForWithCodec(t.Context(), NewProducer(newTestQueue()), "send_email", emailPayload{}, failingCodec{
		marshalErr: wantErr,
	})

	require.ErrorIs(t, err, wantErr)
}

func TestHandleForWithCodecUsesDefaultCodecWhenNil(t *testing.T) {
	t.Parallel()

	var handled emailPayload
	config := newWorkerConfig(HandleForWithCodec("send_email", nil, HandlerFuncFor[emailPayload](func(_ context.Context, task *TaskFor[emailPayload]) error {
		handled = task.Payload
		return nil
	})))
	handler := config.handlers["send_email"]
	require.NotNil(t, handler)

	err := handler.Handle(t.Context(), &Task{Payload: []byte(`{"user_id":40,"subject":"digest"}`)})
	require.NoError(t, err)
	assert.Equal(t, emailPayload{UserID: 40, Subject: "digest"}, handled)
}

func TestHandleForWithCodecReturnsUnmarshalError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("unmarshal failed")
	config := newWorkerConfig(HandleForWithCodec("send_email", failingCodec{
		unmarshalErr: wantErr,
	}, HandlerFuncFor[emailPayload](func(context.Context, *TaskFor[emailPayload]) error {
		return nil
	})))
	handler := config.handlers["send_email"]
	require.NotNil(t, handler)

	err := handler.Handle(t.Context(), &Task{Payload: []byte("invalid")})

	require.ErrorIs(t, err, wantErr)
}

func TestHandleForWithCodecIgnoresNilHandler(t *testing.T) {
	t.Parallel()

	config := newWorkerConfig(HandleForWithCodec[emailPayload]("send_email", nil, nil))

	assert.Empty(t, config.handlers)
}

func TestTaskProducerEnqueueUsesBoundTaskType(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	q := newTestQueue()
	producer := NewTaskProducer[emailPayload](NewProducer(q), "send_email")

	task, err := producer.Enqueue(ctx, emailPayload{
		UserID:  20,
		Subject: "digest",
	}, WithQueue("critical"))
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, "send_email", task.Type)
	assert.Equal(t, "critical", task.Queue)

	lease, err := q.Dequeue(ctx, "critical", time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lease)
	require.NotNil(t, lease.Task())

	var decoded emailPayload
	require.NoError(t, json.Unmarshal(lease.Task().Payload, &decoded))
	assert.Equal(t, 20, decoded.UserID)
	assert.Equal(t, "digest", decoded.Subject)
}
