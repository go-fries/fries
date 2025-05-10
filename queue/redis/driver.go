package redis

import (
	"context"

	"github.com/go-fries/fries/queue/v3"
	"github.com/redis/go-redis/v9"
)

type Driver struct {
	client redis.UniversalClient
	prefix string
}

var _ queue.Driver = (*Driver)(nil)

func NewDriver(client redis.UniversalClient, opts ...Option) *Driver {
	d := &Driver{
		client: client,
		prefix: "fries:",
	}
	for _, opt := range opts {
		opt.apply(d)
	}
	return d
}

func (d *Driver) Enqueue(ctx context.Context, queue string, data []byte) error {
	return d.client.LPush(ctx, d.prefix+queue, data).Err()
}

func (d *Driver) Dequeue(ctx context.Context, queue string) ([]byte, error) {
	data, err := d.client.RPop(ctx, d.prefix+queue).Bytes()
	if err != nil {
		return nil, err
	}
	return data, nil
}
