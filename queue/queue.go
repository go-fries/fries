package queue

import (
	"context"

	"github.com/go-fries/fries/codec/json/v3"
	"github.com/go-fries/fries/codec/v3"
)

type Queue struct {
	driver Driver
	codec  codec.Codec
}

func NewQueue(driver Driver) *Queue {
	return &Queue{
		driver: driver,
		codec:  json.Codec,
	}
}

func (q *Queue) Enqueue(ctx context.Context, queue string, task any) error {
	bytes, err := q.codec.Marshal(task)
	if err != nil {
		return err
	}
	return q.driver.Enqueue(ctx, queue, bytes)
}

func (q *Queue) Dequeue(ctx context.Context, queue string) (any, error) {
	data, err := q.driver.Dequeue(ctx, queue)
	if err != nil {
		return nil, err
	}
	var task any
	if err := q.codec.Unmarshal(data, &task); err != nil {
		return nil, err
	}
	return task, nil
}
