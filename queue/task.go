package queue

import (
	"context"
)

// type Message struct {
// 	ID       string // unique identifier for the message
// 	Queue    string // name of the queue
// 	MaxTries int    // maximum number of attempts to process the message
// 	Payload  []byte // actual data of the message
// }

type Message interface {
	Fire(ctx context.Context) error
	Release(ctx context.Context) error
	Failed(ctx context.Context) error
	Finish(ctx context.Context) error
}

type message[T any] struct {
	queueManager *Queue                                                   // todo: rename queueManager
	id           string                                                   // unique identifier for the task
	queue        string                                                   // name of the queue
	payload      []byte                                                   // actual data of the task
	handler      func(ctx context.Context, message Message, task T) error // function to handle the task
}

var _ Message = (*message[any])(nil)

func (m *message[T]) Fire(ctx context.Context) error {
	var msg T
	if err := m.queueManager.codec.Unmarshal(m.payload, &msg); err != nil {
		return err
	}
	if m.handler == nil {
		return nil
	}
	return func() error {
		if err := m.handler(ctx, m, msg); err != nil {
			_ = m.Release(ctx)
			return err
		}
		return m.Finish(ctx)
	}()
}

func (m *message[T]) Release(ctx context.Context) error {
	// todo: implement release logic
	return nil
}

func (m *message[T]) Failed(ctx context.Context) error {
	// TODO implement me
	panic("implement me")
}

func (m *message[T]) Finish(ctx context.Context) error {
	// TODO implement me
	panic("implement me")
}

// type Task interface {
// 	Fire(ctx context.Context) error
// 	Release(ctx context.Context) error
// }
//
// type TaskHandler[T any] interface {
// 	Handle(ctx context.Context, task T) error
// }

// type Task[T any] struct {
// 	ID      string
// 	Queue   string
// 	Payload []byte
// 	Handler func(ctx context.Context, task T) error
// }
//
// func (t *Task[T]) Fire(ctx context.Context) error {
// 	if t.Handler == nil {
// 		return nil
// 	}
// 	var task T
// 	if err := json.Unmarshal(t.Payload, &task); err != nil { // todo: /??
// 		return err
// 	}
// 	return t.Handler(ctx, task)
// }
//
// func (t *Task[T]) Release(ctx context.Context) error {
//
// }
