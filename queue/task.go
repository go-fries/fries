package queue

import (
	"maps"
	"time"
)

const DefaultQueue = "default"

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

type Lease struct {
	Task  *Task
	Token string
}
