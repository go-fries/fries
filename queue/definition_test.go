package queue

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefinitionEnqueueOptionsAndObserver(t *testing.T) {
	t.Parallel()

	definition := Define[emailPayload]("notifications.send_email.v1")
	assert.Equal(t, "notifications.send_email.v1", definition.TaskType())
	key := contextValueKey("definition")
	backend := newTestQueue()
	q := &contextRecordingQueue{Queue: backend, key: key}
	var events []Event
	observer := ObserverFunc(func(ctx context.Context, event Event) context.Context {
		events = append(events, event)
		if event.Kind == EventEnqueueStarted {
			return context.WithValue(ctx, key, "observed")
		}
		assert.Equal(t, "observed", ctx.Value(key))
		return ctx
	})
	producer := NewProducer(q, WithObserver(observer), WithQueue("default-email"), WithMetadata(map[string]string{"source": "api"}))
	task, err := definition.Enqueue(t.Context(), producer, emailPayload{UserID: 42, Subject: "welcome"},
		WithID("email-42"), WithQueue("mail"), WithDelay(time.Minute), WithMetadataValue("tenant", "acme"))
	require.NoError(t, err)
	assert.Equal(t, "email-42", task.ID)
	assert.Equal(t, definition.TaskType(), task.Type)
	assert.Equal(t, "mail", task.Queue)
	assert.Equal(t, time.Minute, task.AvailableAt.Sub(task.CreatedAt))
	assert.Equal(t, map[string]string{"source": "api", "tenant": "acme"}, task.Metadata)
	assert.JSONEq(t, `{"user_id":42,"subject":"welcome"}`, string(task.Payload))
	assert.Equal(t, "observed", q.value)
	require.Len(t, events, 2)
	assert.Equal(t, EventEnqueueStarted, events[0].Kind)
	assert.Equal(t, EventEnqueued, events[1].Kind)
	assert.Equal(t, definition.TaskType(), events[1].Task.Type)
	assert.Equal(t, time.Minute, events[0].Delay)
	require.Len(t, backend.queues["mail"], 1)
	assert.Equal(t, task, backend.queues["mail"][0])
}

func TestDefinitionEnqueueErrors(t *testing.T) {
	t.Parallel()

	for _, definition := range []Definition[string]{{}, Define[string](""), Define[string]("", WithCodec(failingCodec{marshalErr: assert.AnError}))} {
		assert.Empty(t, definition.TaskType())
		// Invalid definitions fail before nil producer or codec validation.
		task, err := definition.Enqueue(t.Context(), nil, "value")
		require.ErrorIs(t, err, ErrInvalidTaskType)
		assert.Nil(t, task)
	}

	definition := Define[string]("message")
	_, err := definition.Enqueue(t.Context(), nil, "value")
	require.ErrorContains(t, err, "producer is nil")

	observer := &recordingObserver{}
	producer := NewProducer(newTestQueue(), WithObserver(observer))
	badDefinition := Define[string]("message", WithCodec(failingCodec{marshalErr: assert.AnError}))
	_, err = badDefinition.Enqueue(t.Context(), producer, "value")
	require.ErrorIs(t, err, assert.AnError)
	assert.Empty(t, observer.Events())

	producer = NewProducer(enqueueErrorQueue{err: assert.AnError}, WithObserver(observer))
	task, err := definition.Enqueue(t.Context(), producer, "value")
	require.ErrorIs(t, err, assert.AnError)
	assert.Nil(t, task)
	assert.Equal(t, []EventKind{EventEnqueueStarted, EventEnqueueFailed}, observer.Kinds())

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = definition.Enqueue(ctx, NewProducer(newTestQueue()), "value")
	require.ErrorIs(t, err, context.Canceled)
}

func TestDefinitionHandlerCodecsAndEnvelope(t *testing.T) {
	t.Parallel()

	// Reuse options across concurrently constructed, independent definitions.
	rawOption := WithCodec(passthroughCodec{})
	jsonOption := WithCodec(nil)
	for _, tt := range []struct {
		name     string
		opts     []DefinitionOption
		wire     string
		fullTask bool
	}{
		{name: "payload JSON", wire: `"hello"`},
		{name: "payload custom codec", opts: []DefinitionOption{rawOption}, wire: "hello"},
		{name: "full task JSON", wire: `"hello"`, fullTask: true},
		{name: "full task custom codec", opts: []DefinitionOption{rawOption}, wire: "hello", fullTask: true},
		{name: "nil codec selects JSON", opts: []DefinitionOption{jsonOption}, wire: `"hello"`},
		{name: "last codec wins", opts: []DefinitionOption{WithCodec(failingCodec{marshalErr: assert.AnError}), rawOption}, wire: "hello"},
		{name: "nil codec resets to JSON", opts: []DefinitionOption{rawOption, jsonOption}, wire: `"hello"`, fullTask: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			definition := Define[string]("message.v1", tt.opts...)
			q := newTestQueue()
			task, err := definition.Enqueue(ctx, NewProducer(q), "hello",
				WithID("task-1"), WithMetadataValue("tenant", "acme"))
			require.NoError(t, err)
			assert.Equal(t, tt.wire, string(task.Payload))
			delivery, err := q.Receive(ctx, DefaultQueue)
			require.NoError(t, err)

			var calls []string
			checkPayload := func(received context.Context, payload string) {
				assert.Same(t, ctx, received)
				assert.Equal(t, "hello", payload)
				calls = append(calls, "handler")
			}
			option := definition.Handle(func(received context.Context, payload string) error {
				checkPayload(received, payload)
				return nil
			})
			if tt.fullTask {
				option = definition.HandleFor(HandlerFuncFor[string](func(received context.Context, task *TaskFor[string]) error {
					assert.Same(t, delivery.Task(), task.Task)
					assert.Equal(t, "task-1", task.Task.ID)
					assert.Equal(t, 1, task.Task.Attempt)
					assert.Equal(t, "acme", task.Task.Metadata["tenant"])
					checkPayload(received, task.Payload)
					return nil
				}))
			}
			worker := NewWorker(q, option, WithMiddleware(func(next Handler) Handler {
				return HandlerFunc(func(ctx context.Context, task *Task) error {
					calls = append(calls, "before")
					err := next.Handle(ctx, task)
					calls = append(calls, "after")
					return err
				})
			}))
			require.NoError(t, worker.process(ctx, &recordingDelivery{
				task: delivery.Task(),
				ack: func(context.Context) error {
					calls = append(calls, "ack")
					return nil
				},
			}))
			assert.Equal(t, []string{"before", "handler", "after", "ack"}, calls)
		})
	}
}

func TestDefinitionRegistration(t *testing.T) {
	t.Parallel()

	definition := Define[string]("message")
	var zero Definition[string]
	noop := func(context.Context, string) error { return nil }
	full := HandlerFuncFor[string](func(context.Context, *TaskFor[string]) error { return nil })
	config := newWorkerConfig(
		zero.Handle(noop), zero.HandleFor(full),
		Define[string]("").Handle(noop),
		definition.Handle(nil), definition.HandleFor(nil),
	)
	assert.Empty(t, config.handlers)

	var received string
	config = newWorkerConfig(
		definition.Handle(func(context.Context, string) error {
			t.Error("replaced handler called")
			return nil
		}),
		definition.HandleFor(HandlerFuncFor[string](func(_ context.Context, task *TaskFor[string]) error {
			received = task.Payload
			return nil
		})),
		definition.Handle(nil), // Ignored registrations do not remove a handler.
	)
	require.Len(t, config.handlers, 1)
	require.NoError(t, config.handlers[definition.TaskType()].Handle(t.Context(), &Task{Payload: []byte(`"last"`)}))
	assert.Equal(t, "last", received)
}

func TestDefinitionDecodeFailureSkipsHandler(t *testing.T) {
	t.Parallel()

	for _, fullTask := range []bool{false, true} {
		called := false
		badCodec := failingCodec{unmarshalErr: assert.AnError}
		definition := Define[string]("message", WithCodec(badCodec))
		option := definition.Handle(func(context.Context, string) error {
			called = true
			return nil
		})
		if fullTask {
			option = definition.HandleFor(HandlerFuncFor[string](func(context.Context, *TaskFor[string]) error {
				called = true
				return nil
			}))
		}
		observer := &recordingObserver{}
		worker := NewWorker(newTestQueue(), option, WithObserver(observer))
		require.NoError(t, worker.process(t.Context(), &recordingDelivery{
			task: &Task{Type: definition.TaskType(), Attempt: 1},
		}))
		assert.False(t, called)
		assert.Equal(t, []EventKind{EventHandlerStarted, EventHandlerFailed, EventTaskRetried}, observer.Kinds())
		assert.ErrorIs(t, observer.Events()[1].Err, assert.AnError)
	}
}

func TestDefinitionWorkerRoundTrip(t *testing.T) {
	t.Parallel()

	definition := Define[emailPayload]("notifications.send_email.v1")
	q := newTestQueue()
	payload := emailPayload{UserID: 42, Subject: "welcome"}
	_, err := definition.Enqueue(t.Context(), NewProducer(q), payload)
	require.NoError(t, err)

	acked := make(chan struct{})
	observer := ObserverFunc(func(ctx context.Context, event Event) context.Context {
		if event.Kind == EventTaskAcked {
			close(acked)
		}
		return ctx
	})
	var received emailPayload
	worker := NewWorker(q, WithObserver(observer), definition.Handle(func(_ context.Context, value emailPayload) error {
		received = value
		return fmt.Errorf("already processed: %w", ErrDiscard)
	}))
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		defer close(done)
		done <- worker.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		// Run's exit is also joined when an assertion aborts the test.
		for range done {
		}
	})
	// Successful settlement signals that Run has started before Stop is called.
	select {
	case <-acked:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	require.NoError(t, worker.Stop(ctx))
	require.NoError(t, <-done)
	assert.Equal(t, payload, received)
}
