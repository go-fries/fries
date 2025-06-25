package redis

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-fries/fries/codec/sonic/v3"
	"github.com/go-fries/fries/codec/v3"
	"github.com/go-fries/fries/eino/components/embedding/cached/v3"
	"github.com/redis/go-redis/v9"
)

var defaultCodec = sonic.Codec

type Cacher struct {
	rdb    redis.UniversalClient
	prefix string
	codec  codec.Codec
}

type Option interface {
	apply(*Cacher)
}

type optionFunc func(*Cacher)

func (f optionFunc) apply(c *Cacher) {
	f(c)
}

func WithPrefix(prefix string) Option {
	return optionFunc(func(c *Cacher) {
		c.prefix = strings.TrimSuffix(prefix, ":") + ":"
	})
}

func WithCodec(codec codec.Codec) Option {
	return optionFunc(func(c *Cacher) {
		c.codec = codec
	})
}

var _ cached.Cacher = (*Cacher)(nil)

func NewCacher(rdb redis.UniversalClient, opts ...Option) *Cacher {
	cacher := &Cacher{
		rdb:    rdb,
		prefix: "eino:",
		codec:  defaultCodec,
	}
	for _, opt := range opts {
		opt.apply(cacher)
	}
	return cacher
}

func (c *Cacher) Set(ctx context.Context, key string, value []float64, expire time.Duration) error {
	data, err := c.codec.Marshal(value)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, c.prefix+key, data, expire).Err()
}

func (c *Cacher) Get(ctx context.Context, key string) ([]float64, error) {
	data, err := c.rdb.Get(ctx, c.prefix+key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, cached.ErrCacherKeyNotFound
		}
		return nil, err
	}

	var value []float64
	return value, c.codec.Unmarshal(data, &value)
}
