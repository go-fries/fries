package recovery

import (
	"bytes"
	"context"
	"log"
	"strings"
	"sync"
	"testing"

	"github.com/go-fries/fries/queue/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var defaultLoggerMu sync.Mutex

func TestNewRecoversPanic(t *testing.T) {
	t.Parallel()

	task := &queue.Task{ID: "task-1", Type: "send_email"}
	middleware := New(WithHandler(func(_ context.Context, got *queue.Task, recovered any, stack []byte) {
		assert.Same(t, task, got)
		assert.Equal(t, "panic", recovered)
		assert.Contains(t, string(stack), "recovery_test.go")
	}))

	handler := middleware(queue.HandlerFunc(func(context.Context, *queue.Task) error {
		panic("panic")
	}))

	require.NotPanics(t, func() {
		err := handler.Handle(t.Context(), task)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "panic")
	})
}

func TestNewPassesThroughSuccess(t *testing.T) {
	t.Parallel()

	task := &queue.Task{ID: "task-1", Type: "send_email"}
	called := false
	handler := New()(queue.HandlerFunc(func(_ context.Context, got *queue.Task) error {
		called = true
		assert.Same(t, task, got)
		return nil
	}))

	require.NoError(t, handler.Handle(t.Context(), task))
	assert.True(t, called)
}

func TestWithStackSizeIgnoresInvalidSize(t *testing.T) {
	t.Parallel()

	c := newConfig(WithStackSize(0), WithHandler(nil))

	assert.NotNil(t, c.handler)
	assert.Positive(t, c.stackSize)
}

func TestDefaultHandlerDoesNotLogTaskPayload(t *testing.T) {
	defaultLoggerMu.Lock()
	defer defaultLoggerMu.Unlock()

	var out bytes.Buffer
	original := log.Writer()
	log.SetOutput(&out)
	defer log.SetOutput(original)

	DefaultHandler(
		t.Context(),
		&queue.Task{
			ID:      "task-1",
			Type:    "send_email",
			Queue:   "critical",
			Attempt: 2,
			Payload: []byte("secret-payload"),
		},
		"panic",
		[]byte("stack"),
	)

	logged := out.String()
	assert.Contains(t, logged, "task_id=task-1")
	assert.Contains(t, logged, "task_type=send_email")
	assert.NotContains(t, logged, "secret-payload")
	assert.False(t, strings.Contains(logged, "Payload"))
}
