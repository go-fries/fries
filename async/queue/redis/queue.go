package redis

import (
	"context"

	"github.com/go-fries/fries/async/v3"
	"github.com/go-fries/fries/codec/v3"
	"github.com/redis/go-redis/v9"
)

type Queue struct {
	name   string
	client redis.UniversalClient
	codec  codec.Codec
}

var _ async.Queue = (*Queue)(nil)

func New(client redis.UniversalClient, opts ...Option) *Queue {
	c := newConfig(opts...)

	return &Queue{
		name:   c.name,
		client: client,
		codec:  c.codec,
	}
}

func (q *Queue) Enqueue(ctx context.Context, message *async.Message) error {
	bytes, err := q.codec.Marshal(message)
	if err != nil {
		return err
	}

	return q.client.LPush(ctx, q.name, bytes).Err()
}

func (q *Queue) Dequeue(ctx context.Context) (*async.Message, error) {
	result, err := q.client.RPop(ctx, q.name).Bytes()
	if err != nil {
		return nil, err
	}

	var message async.Message
	if err := q.codec.Unmarshal(result, &message); err != nil {
		return nil, err
	}

	return &message, nil
}
