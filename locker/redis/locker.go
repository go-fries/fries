package redis

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/go-fires/fires/v3/locker"
)

// releaseScript is a Lua script to release a lock in an atomic way.
//
//	KEYS[1] is the lock key
//	ARGV[1] is the lock value
const releaseScript = `if redis.call("get",KEYS[1]) == ARGV[1] then
    return redis.call("del",KEYS[1])
else
    return 0
end`

type Locker struct {
	redis redis.UniversalClient
	name  string
	ttl   time.Duration
	sleep time.Duration // for Until
}

type Option func(*Locker)

func WithName(name string) Option {
	return func(l *Locker) {
		l.name = name
	}
}

func WithTTL(ttl time.Duration) Option {
	return func(l *Locker) {
		l.ttl = ttl
	}
}

func WithSleep(sleep time.Duration) Option {
	return func(l *Locker) {
		l.sleep = sleep
	}
}

var _ locker.Locker = (*Locker)(nil)

func NewLocker(redis redis.UniversalClient, opts ...Option) *Locker {
	l := &Locker{
		redis: redis,
		name:  uuid.New().String(),
		ttl:   time.Second * 10,       //nolint:mnd
		sleep: time.Millisecond * 100, //nolint:mnd
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

func (l *Locker) Try(ctx context.Context, fn func()) error {
	owner, err := l.Get(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = owner.Release(ctx)
	}()

	fn()

	return nil
}

func (l *Locker) Until(ctx context.Context, timeout time.Duration, fn func()) error {
	starting := time.Now()
	owner := locker.NewOwner(l)

	for {
		if ok, err := l.acquire(ctx, owner); err != nil {
			return err
		} else if ok {
			break
		}

		if time.Since(starting) >= timeout {
			return locker.ErrTimeout
		}

		time.Sleep(l.sleep)
	}

	defer func() {
		_ = owner.Release(ctx)
	}()

	fn()

	return nil
}

func (l *Locker) Get(ctx context.Context) (locker.Owner, error) {
	owner := locker.NewOwner(l)
	ok, err := l.acquire(ctx, owner)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, locker.ErrLocked
	}
	return owner, nil
}

func (l *Locker) Release(ctx context.Context, owner locker.Owner) error {
	if val, err := l.redis.Eval(ctx, releaseScript, []string{l.name}, owner.Name()).Result(); err != nil {
		return err
	} else if val == int64(0) {
		return locker.ErrNotLocked
	}
	return nil
}

func (l *Locker) ForceRelease(ctx context.Context) error {
	return l.redis.Del(ctx, l.name).Err()
}

func (l *Locker) LockedOwner(ctx context.Context) (locker.Owner, error) {
	val, err := l.redis.Get(ctx, l.name).Result()
	if err != nil {
		return nil, err
	} else if val == "" {
		return nil, locker.ErrNotLocked
	}

	return locker.NewOwner(l, locker.WithOwnerName(val)), nil
}

func (l *Locker) acquire(ctx context.Context, owner locker.Owner) (bool, error) {
	return l.redis.SetNX(ctx, l.name, owner.Name(), l.ttl).Result()
}
