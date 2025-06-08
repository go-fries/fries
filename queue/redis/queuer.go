package redis

import (
	"context"
	"errors"
	"strings"

	"github.com/redis/go-redis/v9"

	"github.com/go-fries/fries/queue/v3"
)

type Queuer struct {
	client redis.UniversalClient
	prefix string
}
type Option func(*Queuer)

func WithPrefix(prefix string) Option {
	return func(q *Queuer) {
		q.prefix = strings.TrimSuffix(prefix, ":") + ":"
	}
}

var _ queue.Queuer = (*Queuer)(nil)

func NewQueuer(client redis.UniversalClient, opts ...Option) *Queuer {
	q := &Queuer{
		client: client,
		prefix: "queue:",
	}
	for _, opt := range opts {
		opt(q)
	}
	return q
}

func (q *Queuer) Enqueue(ctx context.Context, queue string, data []byte) error {
	return q.client.LPush(ctx, q.prefix+queue, data).Err()
}

func (q *Queuer) Dequeue(ctx context.Context, queue string) ([]byte, error) {
	return q.client.RPop(ctx, q.prefix+queue).Bytes()
}

func (q *Queuer) Len(ctx context.Context, queue string) (int64, error) {
	return q.client.LLen(ctx, q.prefix+queue).Result()
}

func (q *Queuer) IsEmpty(ctx context.Context, queue string) (bool, error) {
	length, err := q.Len(ctx, queue)
	if err != nil {
		return false, err
	}
	return length == 0, nil
}

func (q *Queuer) Peek(ctx context.Context, queue string) ([]byte, error) {
	data, err := q.client.LIndex(ctx, q.prefix+queue, 0).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	return data, nil
}

func (q *Queuer) Drain(ctx context.Context, queue string) error {
	return q.client.Del(ctx, q.prefix+queue).Err()
}
