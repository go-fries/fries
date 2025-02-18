package redis

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-fries/fries/cache/v3"
	"github.com/go-fries/fries/codec/json/v3"
	"github.com/go-fries/fries/codec/v3"
	lockerredis "github.com/go-fries/fries/locker/redis/v3"
	"github.com/go-fries/fries/locker/v3"
	"github.com/redis/go-redis/v9"
)

type Store struct {
	prefix string
	codec  codec.Codec
	redis  redis.UniversalClient
}

type Option func(*Store)

func Prefix(prefix string) Option {
	return func(s *Store) {
		if prefix != "" {
			if !strings.HasSuffix(prefix, ":") {
				s.prefix = prefix + ":"
			} else {
				s.prefix = prefix
			}
		}
	}
}

func Codec(codec codec.Codec) Option {
	return func(s *Store) {
		s.codec = codec
	}
}

var (
	_ cache.Store   = (*Store)(nil)
	_ cache.Addable = (*Store)(nil)
)

func New(redis redis.UniversalClient, opts ...Option) *Store {
	story := &Store{
		codec: json.Codec,
		redis: redis,
	}
	for _, o := range opts {
		o(story)
	}
	return story
}

func (s *Store) Has(ctx context.Context, key string) (bool, error) {
	r := s.redis.Exists(ctx, s.prefix+key)
	if r.Err() != nil {
		return false, r.Err()
	}

	return r.Val() > 0, nil
}

func (s *Store) Get(ctx context.Context, key string, dest any) error {
	r := s.redis.Get(ctx, s.prefix+key)
	if r.Err() != nil {
		if errors.Is(r.Err(), redis.Nil) {
			return cache.ErrNotFound
		}
		return r.Err()
	}

	return s.codec.Unmarshal([]byte(r.Val()), dest)
}

func (s *Store) Put(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	valued, err := s.codec.Marshal(value)
	if err != nil {
		return false, err
	}

	r := s.redis.Set(ctx, s.prefix+key, valued, ttl)
	if r.Err() != nil {
		return false, r.Err()
	}

	return r.Val() == "OK", nil
}

func (s *Store) Increment(ctx context.Context, key string, value int) (int, error) {
	r := s.redis.IncrBy(ctx, s.prefix+key, int64(value))
	if r.Err() != nil {
		return 0, r.Err()
	}

	return int(r.Val()), nil
}

func (s *Store) Decrement(ctx context.Context, key string, value int) (int, error) {
	r := s.redis.DecrBy(ctx, s.prefix+key, int64(value))
	if r.Err() != nil {
		return 0, r.Err()
	}

	return int(r.Val()), nil
}

func (s *Store) Forever(ctx context.Context, key string, value any) (bool, error) {
	valued, err := s.codec.Marshal(value)
	if err != nil {
		return false, err
	}

	r := s.redis.Set(ctx, s.prefix+key, valued, redis.KeepTTL)
	if r.Err() != nil {
		return false, r.Err()
	}

	return r.Val() == "OK", nil
}

func (s *Store) Forget(ctx context.Context, key string) (bool, error) {
	r := s.redis.Del(ctx, s.prefix+key)
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
	return s.prefix
}

func (s *Store) Add(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	valued, err := s.codec.Marshal(value)
	if err != nil {
		return false, err
	}

	r := s.redis.SetNX(ctx, s.prefix+key, valued, ttl)
	if r.Err() != nil {
		return false, r.Err()
	}

	return r.Val(), nil
}

func (s *Store) Lock(key string, ttl time.Duration) locker.Locker {
	return lockerredis.NewLocker(s.redis,
		lockerredis.WithName(s.prefix+key),
		lockerredis.WithTTL(ttl),
	)
}
