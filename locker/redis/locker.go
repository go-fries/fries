package redis

import (
	"context"
	"fmt"
	"math/rand/v2"
	"reflect"
	"time"

	"github.com/go-fries/fries/locker/v4"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	releaseScript = redis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("del", KEYS[1])
end
return 0
`)
	refreshScript = redis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("pexpire", KEYS[1], ARGV[2])
end
return 0
`)
)

// Locker creates named locks backed by Redis.
type Locker struct {
	client          redis.UniversalClient
	prefix          string
	minWaitInterval time.Duration
	maxWaitInterval time.Duration
}

var _ locker.Locker = (*Locker)(nil)

// New creates a Redis-backed [Locker].
//
// New panics if client is nil.
func New(client redis.UniversalClient, options ...Option) *Locker {
	if isNilClient(client) {
		panic("locker/redis: nil client")
	}
	c := newConfig(options...)
	return &Locker{
		client:          client,
		prefix:          c.prefix,
		minWaitInterval: c.minWaitInterval,
		maxWaitInterval: c.maxWaitInterval,
	}
}

func isNilClient(client redis.UniversalClient) bool {
	if client == nil {
		return true
	}
	value := reflect.ValueOf(client)
	return value.Kind() == reflect.Pointer && value.IsNil()
}

// Lock creates a named Redis lock with the given lease TTL. The name and TTL
// are validated when the lock is acquired.
func (l *Locker) Lock(name string, ttl time.Duration) locker.Lock {
	return &Lock{
		client:          l.client,
		name:            name,
		key:             l.prefix + name,
		ttl:             ttl,
		minWaitInterval: l.minWaitInterval,
		maxWaitInterval: l.maxWaitInterval,
	}
}

// Lock represents a named Redis lock.
type Lock struct {
	client          redis.UniversalClient
	name            string
	key             string
	ttl             time.Duration
	minWaitInterval time.Duration
	maxWaitInterval time.Duration
}

var (
	_ locker.Lock           = (*Lock)(nil)
	_ locker.RestorableLock = (*Lock)(nil)
)

// TryAcquire performs one acquisition attempt. It returns
// [locker.ErrNotAcquired] when another lease currently owns the lock.
func (l *Lock) TryAcquire(ctx context.Context) (locker.Lease, error) {
	if err := l.validate(ctx); err != nil {
		return nil, err
	}
	return l.tryAcquire(ctx, uuid.NewString())
}

// Acquire waits until the lock is acquired or ctx is done. Backend errors are
// returned immediately instead of being treated as lock contention.
func (l *Lock) Acquire(ctx context.Context) (locker.Lease, error) {
	if err := l.validate(ctx); err != nil {
		return nil, err
	}

	token := uuid.NewString()
	for {
		lease, err := l.tryAcquire(ctx, token)
		if err == nil {
			return lease, nil
		}
		if err != locker.ErrNotAcquired {
			return nil, err
		}
		if err := wait(ctx, l.waitInterval()); err != nil {
			return nil, err
		}
	}
}

// Restore creates a lease for an existing ownership token without accessing
// Redis. Ownership is checked atomically when the lease is released or
// refreshed.
func (l *Lock) Restore(token string) (locker.Lease, error) {
	if l.name == "" {
		return nil, locker.ErrInvalidName
	}
	if token == "" {
		return nil, locker.ErrInvalidToken
	}
	return &Lease{client: l.client, key: l.key, token: token}, nil
}

func (l *Lock) validate(ctx context.Context) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if l.name == "" {
		return locker.ErrInvalidName
	}
	if l.ttl <= 0 {
		return locker.ErrInvalidTTL
	}
	return nil
}

func (l *Lock) tryAcquire(ctx context.Context, token string) (locker.Lease, error) {
	acquired, err := l.client.SetNX(ctx, l.key, token, l.ttl).Result()
	if err != nil {
		return nil, fmt.Errorf("locker/redis: acquire %q: %w", l.key, err)
	}
	if !acquired {
		return nil, locker.ErrNotAcquired
	}
	return &Lease{client: l.client, key: l.key, token: token}, nil
}

func (l *Lock) waitInterval() time.Duration {
	if l.minWaitInterval == l.maxWaitInterval {
		return l.minWaitInterval
	}
	return l.minWaitInterval + time.Duration(
		rand.Int64N(int64(l.maxWaitInterval-l.minWaitInterval)),
	)
}

// Lease represents ownership of a Redis lock.
type Lease struct {
	client redis.UniversalClient
	key    string
	token  string
}

var (
	_ locker.Lease             = (*Lease)(nil)
	_ locker.TransferableLease = (*Lease)(nil)
	_ locker.RenewableLease    = (*Lease)(nil)
)

// Token returns the ownership token for transferring this lease to another
// process. Treat the token as a secret capability.
func (l *Lease) Token() string {
	return l.token
}

// Release releases the lock only if this lease still owns it. It returns
// [locker.ErrLeaseLost] if the lease expired or ownership changed.
func (l *Lease) Release(ctx context.Context) error {
	if err := validateContext(ctx); err != nil {
		return err
	}

	released, err := releaseScript.Run(ctx, l.client, []string{l.key}, l.token).Int64()
	if err != nil {
		return fmt.Errorf("locker/redis: release %q: %w", l.key, err)
	}
	if released == 0 {
		return locker.ErrLeaseLost
	}
	return nil
}

// Refresh resets the lease expiry to ttl from the time Redis processes the
// request. It returns [locker.ErrLeaseLost] if the lease no longer owns the
// lock.
func (l *Lease) Refresh(ctx context.Context, ttl time.Duration) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if ttl <= 0 {
		return locker.ErrInvalidTTL
	}

	refreshed, err := refreshScript.Run(
		ctx,
		l.client,
		[]string{l.key},
		l.token,
		milliseconds(ttl),
	).Int64()
	if err != nil {
		return fmt.Errorf("locker/redis: refresh %q: %w", l.key, err)
	}
	if refreshed == 0 {
		return locker.ErrLeaseLost
	}
	return nil
}

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return locker.ErrInvalidContext
	}
	return ctx.Err()
}

func wait(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func milliseconds(ttl time.Duration) int64 {
	milliseconds := ttl.Milliseconds()
	if milliseconds < 1 {
		return 1
	}
	return milliseconds
}
