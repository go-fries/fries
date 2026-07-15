package redis

import (
	"context"
	"errors"
	"net"
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
		client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
		t.Cleanup(func() { _ = client.Close() })

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
	lock := newLock(client, time.Second)

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
	defaultLock := New(client).Lock(name, time.Second)
	customLock := New(client, WithPrefix("billing:locker:")).Lock(name, time.Second)

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
	name := "locker:test:" + uuid.NewString()
	backend := New(client, WithWaitInterval(5*time.Millisecond, 5*time.Millisecond))
	lock := backend.Lock(name, time.Second)

	first, err := lock.TryAcquire(t.Context())
	require.NoError(t, err)

	released := make(chan error, 1)
	go func() {
		time.Sleep(30 * time.Millisecond)
		released <- first.Release(t.Context())
	}()

	second, err := lock.Acquire(t.Context())
	require.NoError(t, err)
	require.NoError(t, <-released)
	require.NoError(t, second.Release(t.Context()))
}

func TestLockAcquireContext(t *testing.T) {
	client := newRedis(t)
	name := "locker:test:" + uuid.NewString()
	backend := New(client, WithWaitInterval(time.Second, time.Second))
	lock := backend.Lock(name, time.Second)

	lease, err := lock.TryAcquire(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { _ = lease.Release(context.WithoutCancel(t.Context())) })

	ctx, cancel := context.WithTimeout(t.Context(), 40*time.Millisecond)
	defer cancel()
	started := time.Now()

	acquired, err := lock.Acquire(ctx)
	assert.Nil(t, acquired)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(started), 500*time.Millisecond)
}

func TestLockAcquireCanceled(t *testing.T) {
	client := newRedis(t)
	name := "locker:test:" + uuid.NewString()
	backend := New(client, WithWaitInterval(time.Second, time.Second))
	lock := backend.Lock(name, time.Second)

	lease, err := lock.TryAcquire(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { _ = lease.Release(context.WithoutCancel(t.Context())) })

	ctx, cancel := context.WithCancel(t.Context())
	timer := time.AfterFunc(20*time.Millisecond, cancel)
	defer timer.Stop()
	defer cancel()
	acquired, err := lock.Acquire(ctx)
	assert.Nil(t, acquired)
	assert.ErrorIs(t, err, context.Canceled)
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
	lock := newLock(client, 60*time.Millisecond)

	oldLease, err := lock.TryAcquire(t.Context())
	require.NoError(t, err)
	time.Sleep(90 * time.Millisecond)

	newLease, err := lock.TryAcquire(t.Context())
	require.NoError(t, err)
	assert.ErrorIs(t, oldLease.Release(t.Context()), locker.ErrLeaseLost)

	contender, err := lock.TryAcquire(t.Context())
	assert.Nil(t, contender)
	assert.ErrorIs(t, err, locker.ErrNotAcquired)
	require.NoError(t, newLease.Release(t.Context()))
}

func TestLeaseExpired(t *testing.T) {
	client := newRedis(t)
	lock := newLock(client, 40*time.Millisecond)

	lease, err := lock.TryAcquire(t.Context())
	require.NoError(t, err)
	time.Sleep(70 * time.Millisecond)
	assert.ErrorIs(t, lease.Release(t.Context()), locker.ErrLeaseLost)
	assert.ErrorIs(
		t,
		lease.(locker.RenewableLease).Refresh(t.Context(), time.Second),
		locker.ErrLeaseLost,
	)
}

func TestLeaseRefresh(t *testing.T) {
	client := newRedis(t)
	lock := newLock(client, 120*time.Millisecond)

	lease, err := lock.TryAcquire(t.Context())
	require.NoError(t, err)
	renewable := lease.(locker.RenewableLease)

	time.Sleep(80 * time.Millisecond)
	require.NoError(t, renewable.Refresh(t.Context(), 200*time.Millisecond))
	time.Sleep(80 * time.Millisecond)

	contender, err := lock.TryAcquire(t.Context())
	assert.Nil(t, contender)
	assert.ErrorIs(t, err, locker.ErrNotAcquired)

	time.Sleep(150 * time.Millisecond)
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
		lock := newLock(client, time.Second)
		lease, err := lock.TryAcquire(t.Context())
		require.NoError(t, err)

		token := lease.(locker.TransferableLease).Token()
		restored, err := lock.(locker.RestorableLock).Restore(token)
		require.NoError(t, err)
		require.NoError(t, restored.Release(t.Context()))
		assert.ErrorIs(t, lease.Release(t.Context()), locker.ErrLeaseLost)
	})

	t.Run("unknown token", func(t *testing.T) {
		client := newRedis(t)
		lock := newLock(client, time.Second)
		lease, err := lock.TryAcquire(t.Context())
		require.NoError(t, err)
		t.Cleanup(func() { _ = lease.Release(context.WithoutCancel(t.Context())) })

		restored, err := lock.(locker.RestorableLock).Restore("unknown-token")
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
	t.Cleanup(func() { _ = client.Close() })
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

func newRedis(t *testing.T) redis.UniversalClient {
	t.Helper()

	client := redis.NewClient(&redis.Options{
		Addr:        "localhost:6379",
		DialTimeout: 10 * time.Millisecond,
		MaxRetries:  -1,
	})
	if err := client.Ping(t.Context()).Err(); err != nil {
		_ = client.Close()
		t.Skipf("Redis is not available: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func newLock(client redis.UniversalClient, ttl time.Duration) locker.Lock {
	return New(client).Lock("locker:test:"+uuid.NewString(), ttl)
}
