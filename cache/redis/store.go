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
			s.prefix = strings.TrimSuffix(prefix, ":") + ":"
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

var ErrFlushUnsupported = errors.New("cache/redis: prefix flush is not supported by this redis client")

const flushScanCount = 1000

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

	r := s.redis.Set(ctx, s.prefix+key, valued, 0)
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
	flush := func(ctx context.Context, client *redis.Client) error {
		// Prefix flushing uses SCAN followed by DEL, so it is non-atomic and
		// best-effort only. Concurrent writes may add matching keys during or
		// after scanning, and keys may expire between SCAN and DEL. A nil Redis
		// error only means no command failed during this pass.
		var cursor uint64
		for {
			keys, next, err := client.Scan(ctx, cursor, s.prefix+"*", flushScanCount).Result()
			if err != nil {
				return err
			}

			if len(keys) > 0 {
				if err := client.Del(ctx, keys...).Err(); err != nil {
					return err
				}
			}

			cursor = next
			if cursor == 0 {
				break
			}
		}

		return nil
	}

	var err error
	switch client := s.redis.(type) {
	case *redis.Client:
		err = flush(ctx, client)
	case *redis.ClusterClient:
		err = client.ForEachMaster(ctx, flush)
	case *redis.Ring:
		err = client.ForEachShard(ctx, flush)
	default:
		return false, ErrFlushUnsupported
	}

	if err != nil {
		return false, err
	}

	return true, nil
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
	return lockerredis.NewLocker(
		s.redis,
		lockerredis.WithName(s.prefix+key),
		lockerredis.WithTTL(ttl),
	)
}
