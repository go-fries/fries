package redis

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/go-kratos-ecosystem/components/v2/cache"
	"github.com/go-kratos-ecosystem/components/v2/codec"
	"github.com/go-kratos-ecosystem/components/v2/codec/json"
	"github.com/go-kratos-ecosystem/components/v2/locker"
	redislocker "github.com/go-kratos-ecosystem/components/v2/locker/redis"
)

type Store struct {
	redis redis.UniversalClient

	opts *options
}

type options struct {
	prefix string
	codec  codec.Codec
}

type Option func(*options)

func Prefix(prefix string) Option {
	return func(o *options) {
		if prefix != "" {
			if !strings.HasSuffix(prefix, ":") {
				o.prefix = prefix + ":"
			} else {
				o.prefix = prefix
			}
		}
	}
}

func Codec(codec codec.Codec) Option {
	return func(o *options) {
		o.codec = codec
	}
}

var (
	_ cache.Store   = (*Store)(nil)
	_ cache.Addable = (*Store)(nil)
)

func New(redis redis.UniversalClient, opts ...Option) *Store {
	opt := &options{
		codec: json.Codec,
	}

	for _, o := range opts {
		o(opt)
	}

	return &Store{
		redis: redis,
		opts:  opt,
	}
}

func (s *Store) Has(ctx context.Context, key string) (bool, error) {
	r := s.redis.Exists(ctx, s.opts.prefix+key)
	if r.Err() != nil {
		return false, r.Err()
	}

	return r.Val() > 0, nil
}

func (s *Store) Get(ctx context.Context, key string, dest any) error {
	r := s.redis.Get(ctx, s.opts.prefix+key)
	if r.Err() != nil {
		if errors.Is(r.Err(), redis.Nil) {
			return cache.ErrNotFound
		}
		return r.Err()
	}

	return s.opts.codec.Unmarshal([]byte(r.Val()), dest)
}

func (s *Store) Put(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	valued, err := s.opts.codec.Marshal(value)
	if err != nil {
		return false, err
	}

	r := s.redis.Set(ctx, s.opts.prefix+key, valued, ttl)
	if r.Err() != nil {
		return false, r.Err()
	}

	return r.Val() == "OK", nil
}

func (s *Store) Increment(ctx context.Context, key string, value int) (int, error) {
	r := s.redis.IncrBy(ctx, s.opts.prefix+key, int64(value))
	if r.Err() != nil {
		return 0, r.Err()
	}

	return int(r.Val()), nil
}

func (s *Store) Decrement(ctx context.Context, key string, value int) (int, error) {
	r := s.redis.DecrBy(ctx, s.opts.prefix+key, int64(value))
	if r.Err() != nil {
		return 0, r.Err()
	}

	return int(r.Val()), nil
}

func (s *Store) Forever(ctx context.Context, key string, value any) (bool, error) {
	valued, err := s.opts.codec.Marshal(value)
	if err != nil {
		return false, err
	}

	r := s.redis.Set(ctx, s.opts.prefix+key, valued, redis.KeepTTL)
	if r.Err() != nil {
		return false, r.Err()
	}

	return r.Val() == "OK", nil
}

func (s *Store) Forget(ctx context.Context, key string) (bool, error) {
	r := s.redis.Del(ctx, s.opts.prefix+key)
	if r.Err() != nil {
		return false, r.Err()
	}

	return r.Val() > 0, nil
}

func (s *Store) Flush(ctx context.Context) (bool, error) {
	r := s.redis.FlushAll(ctx)
	if r.Err() != nil {
		return false, r.Err()
	}

	return r.Val() == "OK", nil
}

func (s *Store) GetPrefix() string {
	return s.opts.prefix
}

func (s *Store) Add(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	valued, err := s.opts.codec.Marshal(value)
	if err != nil {
		return false, err
	}

	r := s.redis.SetNX(ctx, s.opts.prefix+key, valued, ttl)
	if r.Err() != nil {
		return false, r.Err()
	}

	return r.Val(), nil
}

func (s *Store) Lock(key string, ttl time.Duration) locker.Locker {
	return redislocker.NewLocker(s.redis,
		redislocker.WithName(s.opts.prefix+key),
		redislocker.WithTTL(ttl),
	)
}
