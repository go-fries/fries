package queue

import (
	"context"
)

// TaskFor is a typed view of a task with its payload decoded as T.
type TaskFor[T any] struct {
	// Task is the original queue task with its raw payload and delivery metadata.
	Task *Task

	// Payload is the decoded application payload.
	Payload T
}

// HandlerFor processes a task whose payload has been decoded as T.
type HandlerFor[T any] interface {
	// Handle processes a typed task and returns nil only when it should be acknowledged.
	Handle(ctx context.Context, task *TaskFor[T]) error
}

// Tasker binds a typed handler to the task type it handles.
//
// Use Tasker when an application wants one concrete type to own the task type
// constant and the typed Handle method. Applications can add their own Enqueue
// method to the same concrete type, but Enqueue is intentionally not part of
// this interface so each task can expose a business-friendly enqueue signature.
type Tasker[T any] interface {
	// TaskType returns the queue task type handled by this tasker.
	TaskType() string
	HandlerFor[T]
}

// HandlerFuncFor adapts a function to HandlerFor.
type HandlerFuncFor[T any] func(ctx context.Context, task *TaskFor[T]) error

// Handle calls f(ctx, task).
func (f HandlerFuncFor[T]) Handle(ctx context.Context, task *TaskFor[T]) error {
	return f(ctx, task)
}
