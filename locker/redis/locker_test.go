package redis

import (
	"context"
	"errors"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/go-fries/fries/locker/v4"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("nil client", func(t *testing.T) {
		assert.Panics(t, func() {
			New(nil)
		})

		var client *redis.Client
		assert.Panics(t, func() {
			New(client)
		})
	})

	t.Run("options", func(t *testing.T) {
		client := redis.NewClient(&redis.Options{Addr: "unused:6379"})
		t.Cleanup(func() { assert.NoError(t, client.Close()) })

		backend := New(
			client,
			nil,
			WithPrefix("app:locker:"),
			WithWaitInterval(2*time.Millisecond, 5*time.Millisecond),
		)
		assert.Equal(t, "app:locker:", backend.prefix)
		assert.Equal(t, 2*time.Millisecond, backend.minWaitInterval)
		assert.Equal(t, 5*time.Millisecond, backend.maxWaitInterval)
	})
}

func TestWithPrefix(t *testing.T) {
	tests := map[string]struct {
		option Option
		prefix string
	}{
		"default": {
			prefix: defaultPrefix,
		},
		"custom": {
			option: WithPrefix("billing:locker"),
			prefix: "billing:locker:",
		},
		"trailing colons": {
			option: WithPrefix("billing:locker::"),
			prefix: "billing:locker:",
		},
		"empty": {
			option: WithPrefix(""),
			prefix: defaultPrefix,
		},
		"colons only": {
			option: WithPrefix("::"),
			prefix: defaultPrefix,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.prefix, newConfig(tt.option).prefix)
		})
	}
}

func TestWithWaitInterval(t *testing.T) {
	tests := map[string]struct {
		option  Option
		minimum time.Duration
		maximum time.Duration
	}{
		"default": {
			minimum: defaultMinWaitInterval,
			maximum: defaultMaxWaitInterval,
		},
		"range": {
			option:  WithWaitInterval(time.Millisecond, 2*time.Millisecond),
			minimum: time.Millisecond,
			maximum: 2 * time.Millisecond,
		},
		"fixed": {
			option:  WithWaitInterval(time.Millisecond, time.Millisecond),
			minimum: time.Millisecond,
			maximum: time.Millisecond,
		},
		"zero minimum": {
			option:  WithWaitInterval(0, time.Millisecond),
			minimum: defaultMinWaitInterval,
			maximum: defaultMaxWaitInterval,
		},
		"reversed": {
			option:  WithWaitInterval(2*time.Millisecond, time.Millisecond),
			minimum: defaultMinWaitInterval,
			maximum: defaultMaxWaitInterval,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			c := newConfig(tt.option)
			assert.Equal(t, tt.minimum, c.minWaitInterval)
			assert.Equal(t, tt.maximum, c.maxWaitInterval)
		})
	}
}

func TestLockValidation(t *testing.T) {
	tests := map[string]struct {
		ctx  context.Context
		name string
		ttl  time.Duration
		err  error
	}{
		"nil context": {
			name: "lock",
			ttl:  time.Second,
			err:  locker.ErrInvalidContext,
		},
		"empty name": {
			ctx: t.Context(),
			ttl: time.Second,
			err: locker.ErrInvalidName,
		},
		"zero ttl": {
			ctx:  t.Context(),
			name: "lock",
			err:  locker.ErrInvalidTTL,
		},
		"negative ttl": {
			ctx:  t.Context(),
			name: "lock",
			ttl:  -time.Second,
			err:  locker.ErrInvalidTTL,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			lock := &Lock{name: tt.name, ttl: tt.ttl}

			lease, err := lock.TryAcquire(tt.ctx)
			assert.Nil(t, lease)
			assert.ErrorIs(t, err, tt.err)

			lease, err = lock.Acquire(tt.ctx)
			assert.Nil(t, lease)
			assert.ErrorIs(t, err, tt.err)
		})
	}
}

func TestLockTryAcquire(t *testing.T) {
	client := newRedis(t)
	lock := newLock(t, client, time.Minute)

	first, err := lock.TryAcquire(t.Context())
	require.NoError(t, err)
	require.NotNil(t, first)

	second, err := lock.TryAcquire(t.Context())
	assert.Nil(t, second)
	assert.ErrorIs(t, err, locker.ErrNotAcquired)

	require.NoError(t, first.Release(t.Context()))
	third, err := lock.TryAcquire(t.Context())
	require.NoError(t, err)
	require.NotNil(t, third)
	require.NoError(t, third.Release(t.Context()))
}

func TestRedisKeyPrefix(t *testing.T) {
	client := newRedis(t)
	name := "test:" + uuid.NewString()
	cleanupKeys(t, client, defaultPrefix+name, "billing:locker:"+name)
	defaultLock := New(client).Lock(name, time.Minute)
	customLock := New(client, WithPrefix("billing:locker:")).Lock(name, time.Minute)

	defaultLease, err := defaultLock.TryAcquire(t.Context())
	require.NoError(t, err)
	customLease, err := customLock.TryAcquire(t.Context())
	require.NoError(t, err)

	defaultExists, err := client.Exists(t.Context(), defaultPrefix+name).Result()
	require.NoError(t, err)
	customExists, err := client.Exists(t.Context(), "billing:locker:"+name).Result()
	require.NoError(t, err)
	unprefixedExists, err := client.Exists(t.Context(), name).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), defaultExists)
	assert.Equal(t, int64(1), customExists)
	assert.Zero(t, unprefixedExists)

	require.NoError(t, defaultLease.Release(t.Context()))
	require.NoError(t, customLease.Release(t.Context()))
}

func TestLockAcquireWaitsUntilReleased(t *testing.T) {
	client := newRedis(t)
	lock := newLock(t, client, time.Minute, WithWaitInterval(5*time.Millisecond, 5*time.Millisecond))

	first, err := lock.TryAcquire(t.Context())
	require.NoError(t, err)

	contended := observeContention(client)
	acquired := acquireAsync(t.Context(), t, lock)
	requireContention(t, contended)
	select {
	case result := <-acquired:
		t.Fatalf("Acquire returned before release: %+v", result)
	default:
	}

	require.NoError(t, first.Release(t.Context()))
	result := requireAcquireResult(t, acquired)
	require.NoError(t, result.err)
	require.NotNil(t, result.lease)
	require.NoError(t, result.lease.Release(t.Context()))
}

func TestLockAcquireContext(t *testing.T) {
	client := newRedis(t)
	lock := newLock(t, client, time.Minute, WithWaitInterval(time.Hour, time.Hour))

	_, err := lock.TryAcquire(t.Context())
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 250*time.Millisecond)
	defer cancel()
	contended := observeContention(client)
	acquired := acquireAsync(ctx, t, lock)
	requireContention(t, contended)
	result := requireAcquireResult(t, acquired)
	assert.Nil(t, result.lease)
	assert.ErrorIs(t, result.err, context.DeadlineExceeded)
}

func TestLockAcquireCanceled(t *testing.T) {
	client := newRedis(t)
	lock := newLock(t, client, time.Minute, WithWaitInterval(time.Hour, time.Hour))

	_, err := lock.TryAcquire(t.Context())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	contended := observeContention(client)
	acquired := acquireAsync(ctx, t, lock)
	requireContention(t, contended)
	cancel()
	result := requireAcquireResult(t, acquired)
	assert.Nil(t, result.lease)
	assert.ErrorIs(t, result.err, context.Canceled)
}

func TestLockCanceledContextDoesNotAccessRedis(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	lock := &Lock{name: "lock", ttl: time.Second}

	lease, err := lock.TryAcquire(ctx)
	assert.Nil(t, lease)
	assert.ErrorIs(t, err, context.Canceled)

	lease, err = lock.Acquire(ctx)
	assert.Nil(t, lease)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestLeaseCannotReleaseSuccessor(t *testing.T) {
	client := newRedis(t)
	lock := newLock(t, client, 60*time.Millisecond)

	oldLease, err := lock.TryAcquire(t.Context())
	require.NoError(t, err)
	requireExpired(t, client, lock.key)

	successor := New(client).Lock(lock.name, time.Minute)
	newLease, err := successor.TryAcquire(t.Context())
	require.NoError(t, err)
	assert.ErrorIs(t, oldLease.Release(t.Context()), locker.ErrLeaseLost)

	contender, err := successor.TryAcquire(t.Context())
	assert.Nil(t, contender)
	assert.ErrorIs(t, err, locker.ErrNotAcquired)
	require.NoError(t, newLease.Release(t.Context()))
}

func TestLeaseExpired(t *testing.T) {
	client := newRedis(t)
	lock := newLock(t, client, 40*time.Millisecond)

	lease, err := lock.TryAcquire(t.Context())
	require.NoError(t, err)
	requireExpired(t, client, lock.key)
	assert.ErrorIs(t, lease.Release(t.Context()), locker.ErrLeaseLost)
	assert.ErrorIs(
		t,
		lease.(locker.RenewableLease).Refresh(t.Context(), time.Second),
		locker.ErrLeaseLost,
	)
}

func TestLeaseRefresh(t *testing.T) {
	client := newRedis(t)
	lock := newLock(t, client, time.Minute)

	lease, err := lock.TryAcquire(t.Context())
	require.NoError(t, err)
	renewable := lease.(locker.RenewableLease)

	require.NoError(t, renewable.Refresh(t.Context(), 2*time.Minute))
	ttl, err := client.PTTL(t.Context(), lock.key).Result()
	require.NoError(t, err)
	assert.Greater(t, ttl, time.Minute)
	assert.LessOrEqual(t, ttl, 2*time.Minute)

	contender, err := lock.TryAcquire(t.Context())
	assert.Nil(t, contender)
	assert.ErrorIs(t, err, locker.ErrNotAcquired)

	require.NoError(t, renewable.Refresh(t.Context(), 50*time.Millisecond))
	requireExpired(t, client, lock.key)
	contender, err = lock.TryAcquire(t.Context())
	require.NoError(t, err)
	assert.ErrorIs(t, lease.Release(t.Context()), locker.ErrLeaseLost)
	require.NoError(t, contender.Release(t.Context()))
}

func TestLeaseRefreshValidation(t *testing.T) {
	lease := &Lease{}
	var ctx context.Context

	assert.ErrorIs(t, lease.Refresh(ctx, time.Second), locker.ErrInvalidContext)
	assert.ErrorIs(t, lease.Refresh(t.Context(), 0), locker.ErrInvalidTTL)
	assert.ErrorIs(t, lease.Refresh(t.Context(), -time.Second), locker.ErrInvalidTTL)
}

func TestLockRestore(t *testing.T) {
	t.Run("validation does not access Redis", func(t *testing.T) {
		lock := &Lock{name: "lock"}

		lease, err := lock.Restore("")
		assert.Nil(t, lease)
		assert.ErrorIs(t, err, locker.ErrInvalidToken)

		lease, err = (&Lock{}).Restore("token")
		assert.Nil(t, lease)
		assert.ErrorIs(t, err, locker.ErrInvalidName)

		lease, err = lock.Restore("token")
		require.NoError(t, err)
		assert.Equal(t, "token", lease.(locker.TransferableLease).Token())
	})

	t.Run("transferred token", func(t *testing.T) {
		client := newRedis(t)
		lock := newLock(t, client, time.Minute)
		lease, err := lock.TryAcquire(t.Context())
		require.NoError(t, err)

		token := lease.(locker.TransferableLease).Token()
		restored, err := lock.Restore(token)
		require.NoError(t, err)
		require.NoError(t, restored.Release(t.Context()))
		assert.ErrorIs(t, lease.Release(t.Context()), locker.ErrLeaseLost)
	})

	t.Run("unknown token", func(t *testing.T) {
		client := newRedis(t)
		lock := newLock(t, client, time.Minute)
		_, err := lock.TryAcquire(t.Context())
		require.NoError(t, err)

		restored, err := lock.Restore("unknown-token")
		require.NoError(t, err)
		assert.ErrorIs(t, restored.Release(t.Context()), locker.ErrLeaseLost)
	})
}

func TestLeaseReleaseNilContext(t *testing.T) {
	var ctx context.Context
	assert.ErrorIs(t, (&Lease{}).Release(ctx), locker.ErrInvalidContext)
}

func TestLeaseCanceledContextDoesNotAccessRedis(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	lease := &Lease{}

	assert.ErrorIs(t, lease.Release(ctx), context.Canceled)
	assert.ErrorIs(t, lease.Refresh(ctx, time.Second), context.Canceled)
}

func TestBackendErrorIsNotContention(t *testing.T) {
	backendErr := errors.New("backend failed")
	client := redis.NewClient(&redis.Options{
		Addr: "unused",
		Dialer: func(context.Context, string, string) (net.Conn, error) {
			return nil, backendErr
		},
		MaxRetries: -1,
	})
	t.Cleanup(func() { assert.NoError(t, client.Close()) })
	lock := New(client).Lock("locker:test", time.Second)

	lease, err := lock.Acquire(t.Context())
	assert.Nil(t, lease)
	assert.Error(t, err)
	assert.ErrorIs(t, err, backendErr)
	assert.False(t, errors.Is(err, locker.ErrNotAcquired))
}

func TestWaitInterval(t *testing.T) {
	lock := &Lock{
		minWaitInterval: time.Millisecond,
		maxWaitInterval: 3 * time.Millisecond,
	}
	for range 100 {
		interval := lock.waitInterval()
		assert.GreaterOrEqual(t, interval, time.Millisecond)
		assert.Less(t, interval, 3*time.Millisecond)
	}

	lock.maxWaitInterval = time.Millisecond
	assert.Equal(t, time.Millisecond, lock.waitInterval())
}

func newRedis(t *testing.T) *redis.Client {
	t.Helper()
	if testing.Short() {
		t.Skip("Redis integration test")
	}

	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	client := redis.NewClient(&redis.Options{
		Addr:                  addr,
		DialTimeout:           time.Second,
		ReadTimeout:           time.Second,
		WriteTimeout:          time.Second,
		ContextTimeoutEnabled: true,
		MaxRetries:            -1,
	})
	t.Cleanup(func() { assert.NoError(t, client.Close()) })
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	require.NoError(t, client.Ping(ctx).Err(), "Redis is unavailable at %s", addr)
	return client
}

func newLock(t *testing.T, client *redis.Client, ttl time.Duration, opts ...Option) *Lock {
	t.Helper()
	lock := New(client, opts...).Lock("locker:test:"+uuid.NewString(), ttl).(*Lock)
	cleanupKeys(t, client, lock.key)
	return lock
}

func cleanupKeys(t *testing.T, client *redis.Client, keys ...string) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 3*time.Second)
		defer cancel()
		assert.NoError(t, client.Del(ctx, keys...).Err())
	})
}

func requireExpired(t *testing.T, client *redis.Client, key string) {
	t.Helper()
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		exists, err := client.Exists(t.Context(), key).Result()
		assert.NoError(c, err)
		assert.Zero(c, exists)
	}, 3*time.Second, 10*time.Millisecond)
}

type acquireResult struct {
	lease locker.Lease
	err   error
}

func acquireAsync(ctx context.Context, t *testing.T, lock locker.Lock) <-chan acquireResult {
	t.Helper()
	ctx, cancel := context.WithCancel(ctx)
	result := make(chan acquireResult, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		lease, err := lock.Acquire(ctx)
		result <- acquireResult{lease: lease, err: err}
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("Acquire goroutine did not stop")
		}
	})
	return result
}

func requireAcquireResult(t *testing.T, acquired <-chan acquireResult) acquireResult {
	t.Helper()
	select {
	case result := <-acquired:
		return result
	case <-time.After(3 * time.Second):
		t.Fatal("Acquire did not return")
		return acquireResult{}
	}
}

func requireContention(t *testing.T, contended <-chan struct{}) {
	t.Helper()
	select {
	case <-contended:
	case <-time.After(3 * time.Second):
		t.Fatal("Acquire did not observe Redis lock contention")
	}
}

type contentionHook struct {
	once      sync.Once
	contended chan struct{}
}

func observeContention(client *redis.Client) <-chan struct{} {
	hook := &contentionHook{contended: make(chan struct{})}
	client.AddHook(hook)
	return hook.contended
}

func (h *contentionHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *contentionHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		err := next(ctx, cmd)
		if result, ok := cmd.(*redis.BoolCmd); ok && cmd.Name() == "set" && err == nil && !result.Val() {
			h.once.Do(func() { close(h.contended) })
		}
		return err
	}
}

func (h *contentionHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
