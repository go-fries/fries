package queue

import (
	"maps"
	"time"
)

// DefaultQueue is the queue name used when no queue is explicitly configured.
const DefaultQueue = "default"

// Task is the durable envelope stored and delivered by queue implementations.
type Task struct {
	ID             string            `json:"id"`
	Type           string            `json:"type"`
	Queue          string            `json:"queue"`
	Payload        []byte            `json:"payload"`
	Headers        map[string]string `json:"headers,omitempty"`
	IdempotencyKey string            `json:"idempotency_key,omitempty"`
	Attempt        int               `json:"attempt"`
	CreatedAt      time.Time         `json:"created_at"`
	AvailableAt    time.Time         `json:"available_at"`
}

func (t *Task) clone() *Task {
	return t.Clone()
}

// Clone returns a deep copy of the task payload and headers.
func (t *Task) Clone() *Task {
	if t == nil {
		return nil
	}

	cloned := *t
	if t.Payload != nil {
		cloned.Payload = append([]byte(nil), t.Payload...)
	}
	if t.Headers != nil {
		cloned.Headers = make(map[string]string, len(t.Headers))
		maps.Copy(cloned.Headers, t.Headers)
	}
	return &cloned
}

// Lease represents a task delivery that can be acknowledged, retried, or dead-lettered.
type Lease struct {
	Task  *Task
	Token string
}
