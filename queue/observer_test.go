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
	mu     sync.Mutex
	events []Event
}

func (o *recordingObserver) ObserveQueue(_ context.Context, event Event) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.events = append(o.events, event)
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

func hasEventKind(events []Event, kind EventKind) bool {
	for _, event := range events {
		if event.Kind == kind {
			return true
		}
	}
	return false
}
