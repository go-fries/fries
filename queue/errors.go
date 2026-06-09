package queue

import "errors"

var (
	// ErrNoTask is returned by queue implementations when no task is immediately available.
	ErrNoTask = errors.New("queue: no task available")
	// ErrConsumerClosed is returned by Consumer.Receive after the consumer is closed.
	ErrConsumerClosed = errors.New("queue: consumer closed")
	// ErrInvalidTaskType is returned when enqueueing a task without a type.
	ErrInvalidTaskType = errors.New("queue: task type is required")
	// ErrHandlerNotFound is used when a worker receives a task with no registered handler.
	ErrHandlerNotFound = errors.New("queue: handler not found")
)
