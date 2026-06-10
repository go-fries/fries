package queue

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingObserver struct {
	mu       sync.Mutex
	events   []Event
	contexts []context.Context
}

func (o *recordingObserver) ObserveQueue(ctx context.Context, event Event) context.Context {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.events = append(o.events, event)
	o.contexts = append(o.contexts, ctx)
	return ctx
}

func (o *recordingObserver) Events() []Event {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]Event(nil), o.events...)
}

func (o *recordingObserver) Kinds() []EventKind {
	events := o.Events()
	kinds := make([]EventKind, 0, len(events))
	for _, event := range events {
		kinds = append(kinds, event.Kind)
	}
	return kinds
}

func (o *recordingObserver) Contexts() []context.Context {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]context.Context(nil), o.contexts...)
}

func TestProducer_ObserverEvents(t *testing.T) {
	t.Parallel()

	observer := &recordingObserver{}
	producer := NewProducer(newTestQueue(), WithObserver(observer))

	task, err := producer.Enqueue(
		t.Context(),
		"send_email",
		[]byte("secret-payload"),
		WithQueue("critical"),
		WithDelay(time.Second),
	)

	require.NoError(t, err)
	events := observer.Events()
	require.Len(t, events, 2)
	assert.Equal(t, []EventKind{EventEnqueueStarted, EventEnqueued}, observer.Kinds())
	assert.Equal(t, "critical", events[0].Queue)
	assert.Equal(t, TaskInfo{
		ID:    task.ID,
		Type:  "send_email",
		Queue: "critical",
	}, events[0].Task)
	assert.Equal(t, time.Second, events[0].Delay)
	assert.NotEqual(t, string(task.Payload), events[0].Task.ID)
}

func TestProducer_ObserverPassesLifecycleContext(t *testing.T) {
	t.Parallel()

	key := contextValueKey("producer-lifecycle")
	q := &contextRecordingQueue{Queue: newTestQueue(), key: key}
	var enqueuedValue any
	observer := ObserverFunc(func(ctx context.Context, event Event) context.Context {
		switch event.Kind {
		case EventEnqueueStarted:
			return context.WithValue(ctx, key, "enqueue")
		case EventEnqueued:
			enqueuedValue = ctx.Value(key)
		}
		return ctx
	})
	producer := NewProducer(q, WithObserver(observer))

	_, err := producer.Enqueue(t.Context(), "send_email", nil)

	require.NoError(t, err)
	assert.Equal(t, "enqueue", q.value)
	assert.Equal(t, "enqueue", enqueuedValue)
}

func TestProducer_ObserverNilContextFallsBackToOriginalContext(t *testing.T) {
	t.Parallel()

	key := contextValueKey("producer-original")
	q := &contextRecordingQueue{Queue: newTestQueue(), key: key}
	var enqueuedValue any
	observer := ObserverFunc(func(ctx context.Context, event Event) context.Context {
		switch event.Kind {
		case EventEnqueueStarted:
			return nil
		case EventEnqueued:
			enqueuedValue = ctx.Value(key)
		}
		return ctx
	})
	producer := NewProducer(q, WithObserver(observer))
	ctx := context.WithValue(t.Context(), key, "original")

	_, err := producer.Enqueue(ctx, "send_email", nil)

	require.NoError(t, err)
	assert.Equal(t, "original", q.value)
	assert.Equal(t, "original", enqueuedValue)
}

func TestProducer_ObserverFailureEvent(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("enqueue failed")
	observer := &recordingObserver{}
	producer := NewProducer(enqueueErrorQueue{err: wantErr}, WithObserver(observer))

	task, err := producer.Enqueue(t.Context(), "send_email", nil, WithID("task-1"))

	require.ErrorIs(t, err, wantErr)
	assert.Nil(t, task)
	events := observer.Events()
	require.Len(t, events, 2)
	assert.Equal(t, []EventKind{EventEnqueueStarted, EventEnqueueFailed}, observer.Kinds())
	assert.ErrorIs(t, events[1].Err, wantErr)
}

func TestWorker_ObserverEventsForSuccessfulTask(t *testing.T) {
	t.Parallel()

	observer := &recordingObserver{}
	worker := NewWorker(
		newTestQueue(),
		WithQueue("critical"),
		WithConsumerName("worker-1"),
		WithObserver(observer),
		Handle("send_email", HandlerFunc(func(context.Context, *Task) error {
			return nil
		})),
	)
	delivery := &recordingDelivery{
		task: &Task{
			ID:      "task-1",
			Type:    "send_email",
			Queue:   "critical",
			Attempt: 1,
			Payload: []byte("secret-payload"),
		},
	}

	err := worker.process(t.Context(), delivery)

	require.NoError(t, err)
	events := observer.Events()
	require.Len(t, events, 3)
	assert.Equal(t, []EventKind{
		EventHandlerStarted,
		EventHandlerSucceeded,
		EventTaskAcked,
	}, observer.Kinds())
	assert.Equal(t, "critical", events[0].Queue)
	assert.Equal(t, "worker-1", events[0].ConsumerName)
	assert.Equal(t, TaskInfo{
		ID:      "task-1",
		Type:    "send_email",
		Queue:   "critical",
		Attempt: 1,
	}, events[0].Task)
}

func TestWorker_ObserverPassesHandlerLifecycleContext(t *testing.T) {
	t.Parallel()

	key := contextValueKey("handler-lifecycle")
	var handlerValue any
	var succeededValue any
	observer := ObserverFunc(func(ctx context.Context, event Event) context.Context {
		switch event.Kind {
		case EventHandlerStarted:
			return context.WithValue(ctx, key, "handler")
		case EventHandlerSucceeded:
			succeededValue = ctx.Value(key)
		}
		return ctx
	})
	worker := NewWorker(
		newTestQueue(),
		WithObserver(observer),
		Handle("send_email", HandlerFunc(func(ctx context.Context, _ *Task) error {
			handlerValue = ctx.Value(key)
			return nil
		})),
	)
	delivery := &recordingDelivery{
		task: &Task{ID: "task-1", Type: "send_email"},
	}

	err := worker.process(t.Context(), delivery)

	require.NoError(t, err)
	assert.Equal(t, "handler", handlerValue)
	assert.Equal(t, "handler", succeededValue)
}

func TestWorker_ObserverNilContextFallsBackToOriginalContext(t *testing.T) {
	t.Parallel()

	key := contextValueKey("handler-original")
	var handlerValue any
	var succeededValue any
	observer := ObserverFunc(func(ctx context.Context, event Event) context.Context {
		switch event.Kind {
		case EventHandlerStarted:
			return nil
		case EventHandlerSucceeded:
			succeededValue = ctx.Value(key)
		}
		return ctx
	})
	worker := NewWorker(
		newTestQueue(),
		WithObserver(observer),
		Handle("send_email", HandlerFunc(func(ctx context.Context, _ *Task) error {
			handlerValue = ctx.Value(key)
			return nil
		})),
	)
	delivery := &recordingDelivery{
		task: &Task{ID: "task-1", Type: "send_email"},
	}
	ctx := context.WithValue(t.Context(), key, "original")

	err := worker.process(ctx, delivery)

	require.NoError(t, err)
	assert.Equal(t, "original", handlerValue)
	assert.Equal(t, "original", succeededValue)
}

func TestWorker_ObserverEventsForSettlementFailure(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("ack failed")
	observer := &recordingObserver{}
	worker := NewWorker(
		newTestQueue(),
		WithObserver(observer),
		Handle("send_email", HandlerFunc(func(context.Context, *Task) error {
			return nil
		})),
	)
	delivery := &recordingDelivery{
		task: &Task{ID: "task-1", Type: "send_email"},
		ack: func(context.Context) error {
			return wantErr
		},
	}

	err := worker.process(t.Context(), delivery)

	require.ErrorIs(t, err, wantErr)
	events := observer.Events()
	require.Len(t, events, 3)
	assert.Equal(t, EventTaskSettlementFailed, events[2].Kind)
	assert.Equal(t, SettlementAck, events[2].Settlement)
	assert.ErrorIs(t, events[2].Err, wantErr)
}

func TestWorker_ObserverEventsForRunLifecycleAndReceive(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	q := newTestQueue()
	_, err := NewProducer(q).Enqueue(t.Context(), "send_email", nil)
	require.NoError(t, err)

	observer := &recordingObserver{}
	worker := NewWorker(
		q,
		WithObserver(observer),
		Handle("send_email", HandlerFunc(func(context.Context, *Task) error {
			return nil
		})),
	)

	errs := make(chan error, 1)
	go func() {
		errs <- worker.Run(ctx)
	}()

	require.Eventually(t, func() bool {
		return hasEventKind(observer.Events(), EventTaskAcked)
	}, time.Second, time.Millisecond)
	cancel()
	require.NoError(t, <-errs)

	events := observer.Events()
	assert.True(t, hasEventKind(events, EventWorkerStarted))
	assert.True(t, hasEventKind(events, EventTaskReceived))
	assert.True(t, hasEventKind(events, EventWorkerStopped))
}

type enqueueErrorQueue struct {
	err error
}

func (q enqueueErrorQueue) Enqueue(context.Context, *Task) error {
	return q.err
}

func (q enqueueErrorQueue) NewConsumer(context.Context, ConsumerConfig) (Consumer, error) {
	return nil, q.err
}

type contextValueKey string

type contextRecordingQueue struct {
	Queue
	key   any
	value any
}

func (q *contextRecordingQueue) Enqueue(ctx context.Context, task *Task) error {
	q.value = ctx.Value(q.key)
	return q.Queue.Enqueue(ctx, task)
}

func hasEventKind(events []Event, kind EventKind) bool {
	for _, event := range events {
		if event.Kind == kind {
			return true
		}
	}
	return false
}
