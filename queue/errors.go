package queue

import "errors"

var (
	// ErrNoTask is returned by queue implementations when no task is immediately available.
	//
	// The default worker uses blocking Consumer.Receive calls and does not rely on
	// ErrNoTask. Queue implementations may still use it internally or in tests
	// when adapting a non-blocking backend operation.
	ErrNoTask = errors.New("queue: no task available")
	// ErrConsumerClosed is returned by Consumer.Receive after the consumer is closed.
	ErrConsumerClosed = errors.New("queue: consumer closed")
	// ErrInvalidTaskType is returned when enqueueing a task without a type.
	ErrInvalidTaskType = errors.New("queue: task type is required")
	// ErrHandlerNotFound is used when a worker receives a task with no registered handler.
	ErrHandlerNotFound = errors.New("queue: handler not found")
)
