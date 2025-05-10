package queue

import (
	"context"

	"github.com/go-fries/fries/codec/json/v3"
	"github.com/go-fries/fries/codec/v3"
)

type Queue struct {
	driver    Driver
	codec     codec.Codec
	generator Generator
}

func NewQueue(driver Driver) *Queue {
	return &Queue{
		driver:    driver,
		codec:     json.Codec,
		generator: UUIDGenerator(),
	}
}

func (q *Queue) Enqueue(ctx context.Context, queue string, task any) error {
	taskBytes, err := q.codec.Marshal(task)
	if err != nil {
		return err
	}

	msg := Message{
		ID:      q.generator.Generate(),
		Queue:   queue,
		Payload: taskBytes,
	}
	msgBytes, err := q.codec.Marshal(msg)
	if err != nil {
		return err
	}

	return q.driver.Enqueue(ctx, queue, msgBytes)
}

func (q *Queue) Start(ctx context.Context, queue string, fn func(ctx context.Context, msg Message) error) error {
	data, err := q.driver.Dequeue(ctx, queue)
	if err != nil {
		return err
	}
	var msg Message
	if err := q.codec.Unmarshal(data, &msg); err != nil {
		return err
	}
	return fn(ctx, msg)
}
