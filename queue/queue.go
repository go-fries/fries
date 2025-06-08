package queue

import (
	"context"

	"github.com/go-fries/fries/codec/v3"
)

type Queue struct {
	queuer Queuer
	codec  codec.Codec
}

func NewQueue(queuer Queuer) *Queue {
	return &Queue{
		queuer: queuer,
	}
}

func (q *Queue) Push(ctx context.Context, subject string, job Job, opts ...JobOption) error {
	bytes, err := q.codec.Marshal(job)
	if err != nil {
		return err
	}

	j := newJob(subject, bytes, opts...)
	return q.queuer.Enqueue(ctx, j.queue, j.data)
}
