package queue

import "errors"

var (
	// ErrNoTask is returned when a queue has no task available for dequeue.
	ErrNoTask = errors.New("queue: no task available")
	// ErrInvalidTaskType is returned when enqueueing a task without a type.
	ErrInvalidTaskType = errors.New("queue: task type is required")
	// ErrHandlerNotFound is used when a worker receives a task with no registered handler.
	ErrHandlerNotFound = errors.New("queue: handler not found")
)
