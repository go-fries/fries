package queue

import (
	"context"
	"time"
)

// EventKind identifies a queue observer event.
type EventKind string

const (
	// EventEnqueueStarted is emitted before a producer stores a task.
	EventEnqueueStarted EventKind = "enqueue.started"
	// EventEnqueued is emitted after a producer stores a task successfully.
	EventEnqueued EventKind = "enqueue.succeeded"
	// EventEnqueueFailed is emitted when a producer fails to store a task.
	EventEnqueueFailed EventKind = "enqueue.failed"
	// EventWorkerStarted is emitted after a worker starts.
	EventWorkerStarted EventKind = "worker.started"
	// EventWorkerStopped is emitted when a worker exits.
	EventWorkerStopped EventKind = "worker.stopped"
	// EventTaskReceived is emitted after a worker receives a task delivery.
	EventTaskReceived EventKind = "task.received"
	// EventHandlerStarted is emitted before a task handler is called.
	EventHandlerStarted EventKind = "handler.started"
	// EventHandlerSucceeded is emitted when a task handler returns nil.
	EventHandlerSucceeded EventKind = "handler.succeeded"
	// EventHandlerFailed is emitted when a task handler returns a non-nil error.
	EventHandlerFailed EventKind = "handler.failed"
	// EventTaskAcked is emitted after a task delivery is acknowledged.
	EventTaskAcked EventKind = "task.acked"
	// EventTaskRetried is emitted after a task delivery is scheduled for retry.
	EventTaskRetried EventKind = "task.retried"
	// EventTaskDeadLettered is emitted after a task delivery is moved to dead-letter storage.
	EventTaskDeadLettered EventKind = "task.dead_lettered"
	// EventTaskSettlementFailed is emitted when Ack, Retry, or DeadLetter fails.
	EventTaskSettlementFailed EventKind = "task.settlement_failed"
)

// SettlementAction identifies the delivery settlement operation being observed.
type SettlementAction string

const (
	// SettlementAck identifies an Ack operation.
	SettlementAck SettlementAction = "ack"
	// SettlementRetry identifies a Retry operation.
	SettlementRetry SettlementAction = "retry"
	// SettlementDeadLetter identifies a DeadLetter operation.
	SettlementDeadLetter SettlementAction = "dead_letter"
)

// TaskInfo contains low-sensitivity task fields for observer events.
//
// Task payload and metadata are intentionally omitted to avoid accidental
// logging of sensitive business data.
type TaskInfo struct {
	ID      string
	Type    string
	Queue   string
	Attempt int
}

// Event describes a producer or worker event.
type Event struct {
	Kind         EventKind
	Queue        string
	ConsumerName string
	Task         TaskInfo
	Settlement   SettlementAction
	Delay        time.Duration
	Reason       string
	Err          error
}

// Observer receives queue producer and worker events.
//
// Observer implementations should return quickly and avoid blocking queue
// processing. Started events may return a lifecycle context used by later
// events in the same operation. Completion events may also return a context
// used by following settlement events. Returning nil keeps using the input
// context. The default observer is nil and emits no events.
type Observer interface {
	ObserveQueue(ctx context.Context, event Event) context.Context
}

// ObserverFunc adapts a function to Observer.
type ObserverFunc func(ctx context.Context, event Event) context.Context

// ObserveQueue calls f(ctx, event).
func (f ObserverFunc) ObserveQueue(ctx context.Context, event Event) context.Context {
	if f != nil {
		return f(ctx, event)
	}
	return ctx
}

func taskInfo(task *Task) TaskInfo {
	if task == nil {
		return TaskInfo{}
	}
	return TaskInfo{
		ID:      task.ID,
		Type:    task.Type,
		Queue:   task.Queue,
		Attempt: task.Attempt,
	}
}
