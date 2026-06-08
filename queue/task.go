package queue

import (
	"context"
	"maps"
	"time"
)

// DefaultQueue is the queue name used when no queue is explicitly configured.
const DefaultQueue = "default"

// Task is the durable envelope stored and delivered by queue implementations.
type Task struct {
	// ID uniquely identifies the task. Producers generate an ID unless WithID is used.
	ID string `json:"id"`

	// Type identifies the task handler that should process the task.
	Type string `json:"type"`

	// Queue is the logical queue name. Empty values are treated as DefaultQueue.
	Queue string `json:"queue"`

	// Payload is the raw encoded task payload stored by queue implementations.
	//
	// Typed helpers such as EnqueueFor and HandleFor encode and decode this
	// field while exposing the decoded value through TaskFor.Payload.
	Payload []byte `json:"payload"`

	// Metadata carries optional task metadata for handlers and middleware.
	//
	// Metadata is task data and may be persisted, retried, and dead-lettered with
	// the task. Queue delivery state is intentionally kept outside this map.
	Metadata map[string]string `json:"metadata,omitempty"`

	// Attempt is the delivery attempt count observed by the handler.
	//
	// Queue implementations increment it before delivery, so the first handler
	// invocation sees Attempt equal to 1.
	Attempt int `json:"attempt"`

	// CreatedAt is the UTC time when the producer created the task.
	CreatedAt time.Time `json:"created_at"`

	// AvailableAt is the earliest UTC time when the task may be delivered.
	AvailableAt time.Time `json:"available_at"`
}

func (t *Task) clone() *Task {
	return t.Clone()
}

// Clone returns a deep copy of the task payload and metadata.
func (t *Task) Clone() *Task {
	if t == nil {
		return nil
	}

	cloned := *t
	if t.Payload != nil {
		cloned.Payload = append([]byte(nil), t.Payload...)
	}
	if t.Metadata != nil {
		cloned.Metadata = make(map[string]string, len(t.Metadata))
		maps.Copy(cloned.Metadata, t.Metadata)
	}
	return &cloned
}

type noopDelivery struct {
	task *Task
}

// NewDelivery creates a basic delivery for tests and queue implementations that
// do not need backend-specific acknowledgement state.
func NewDelivery(task *Task) Delivery {
	return &noopDelivery{task: task}
}

func (d *noopDelivery) Task() *Task {
	if d == nil {
		return nil
	}
	return d.task
}

func (*noopDelivery) Ack(ctx context.Context) error {
	return ctx.Err()
}

func (*noopDelivery) Retry(ctx context.Context, _ time.Duration) error {
	return ctx.Err()
}

func (*noopDelivery) DeadLetter(ctx context.Context, _ string) error {
	return ctx.Err()
}
