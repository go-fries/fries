package queue

import "errors"

var (
	ErrNoTask          = errors.New("queue: no task available")
	ErrInvalidTaskType = errors.New("queue: task type is required")
	ErrHandlerNotFound = errors.New("queue: handler not found")
)
