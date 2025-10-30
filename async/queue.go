package async

import "context"

type Queue interface {
	Enqueue(ctx context.Context, message *Message) error
	Dequeue(ctx context.Context) (*Message, error)
}

type NoopQueue struct{}

var _ Queue = (*NoopQueue)(nil)

func (q *NoopQueue) Enqueue(context.Context, *Message) error   { return nil }
func (q *NoopQueue) Dequeue(context.Context) (*Message, error) { return nil, nil }
