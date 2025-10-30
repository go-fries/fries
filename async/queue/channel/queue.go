package channel

import (
	"context"

	"github.com/go-fries/fries/async/v3"
)

type Queue struct {
	ch chan *async.Message
}

var _ async.Queue = (*Queue)(nil)

func New(opts ...Option) *Queue {
	c := newConfig(opts...)

	return &Queue{
		ch: make(chan *async.Message, c.size),
	}
}

func (q *Queue) Enqueue(ctx context.Context, message *async.Message) error {
	select {
	case q.ch <- message:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (q *Queue) Dequeue(ctx context.Context) (*async.Message, error) {
	select {
	case msg := <-q.ch:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
