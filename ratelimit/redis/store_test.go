package redis

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-fries/fries/ratelimit/v4"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testPrefixSequence atomic.Uint64

func TestStoreBurstAndRecovery(t *testing.T) {
	store, client := newTestStore(t)
	cleanupKeys(t, client, store, "key")
	request := ratelimit.TakeRequest{
		Key:   "key",
		Limit: ratelimit.Limit{Rate: 1, Period: time.Second, Burst: 3},
		Cost:  1,
	}

	for remaining := 2; remaining >= 0; remaining-- {
		decision, err := store.Take(t.Context(), request)
		require.NoError(t, err)
		assert.True(t, decision.Allowed)
		assert.Equal(t, remaining, decision.Remaining)
		assert.Zero(t, decision.RetryAfter)
	}

	decision, err := store.Take(t.Context(), request)
	require.NoError(t, err)
	assert.False(t, decision.Allowed)
	assert.Zero(t, decision.Remaining)
	assert.Positive(t, decision.RetryAfter)
	assert.LessOrEqual(t, decision.RetryAfter, time.Second)
	assert.Positive(t, decision.ResetAfter)
	assert.LessOrEqual(t, decision.ResetAfter, 3*time.Second)

	time.Sleep(decision.RetryAfter + 10*time.Millisecond)
	decision, err = store.Take(t.Context(), request)
	require.NoError(t, err)
	assert.True(t, decision.Allowed)
}

func TestStoreAllowNIsAllOrNothing(t *testing.T) {
	store, client := newTestStore(t)
	cleanupKeys(t, client, store, "key")
	limit := ratelimit.Limit{Rate: 1, Period: time.Minute, Burst: 4}

	decision, err := store.Take(t.Context(), ratelimit.TakeRequest{
		Key:   "key",
		Limit: limit,
		Cost:  3,
	})
	require.NoError(t, err)
	assert.True(t, decision.Allowed)
	assert.Equal(t, 1, decision.Remaining)

	decision, err = store.Take(t.Context(), ratelimit.TakeRequest{
		Key:   "key",
		Limit: limit,
		Cost:  2,
	})
	require.NoError(t, err)
	assert.False(t, decision.Allowed)
	assert.Equal(t, 1, decision.Remaining)

	decision, err = store.Take(t.Context(), ratelimit.TakeRequest{
		Key:   "key",
		Limit: limit,
		Cost:  1,
	})
	require.NoError(t, err)
	assert.True(t, decision.Allowed)
	assert.Zero(t, decision.Remaining)
}

func TestStoreReset(t *testing.T) {
	store, client := newTestStore(t)
	cleanupKeys(t, client, store, "key")
	request := ratelimit.TakeRequest{
		Key:   "key",
		Limit: ratelimit.Limit{Rate: 1, Period: time.Minute, Burst: 1},
		Cost:  1,
	}

	decision, err := store.Take(t.Context(), request)
	require.NoError(t, err)
	assert.True(t, decision.Allowed)
	decision, err = store.Take(t.Context(), request)
	require.NoError(t, err)
	assert.False(t, decision.Allowed)

	require.NoError(t, store.Reset(t.Context(), request.Key))
	decision, err = store.Take(t.Context(), request)
	require.NoError(t, err)
	assert.True(t, decision.Allowed)
}

func TestStorePrefixesAndKeysAreIsolated(t *testing.T) {
	first, client := newTestStore(t)
	second := New(client, WithPrefix(first.prefix+"other"))
	keys := []string{"key", "binary:\x00:key"}
	for _, store := range []*Store{first, second} {
		cleanupKeys(t, client, store, keys...)
		for _, key := range keys {
			decision, err := store.Take(t.Context(), ratelimit.TakeRequest{
				Key:   key,
				Limit: ratelimit.PerMinute(1),
				Cost:  1,
			})
			require.NoError(t, err)
			assert.True(t, decision.Allowed)
		}
	}
}

func TestStoreExpiresRecoveredKey(t *testing.T) {
	store, client := newTestStore(t)
	cleanupKeys(t, client, store, "key")
	_, err := store.Take(t.Context(), ratelimit.TakeRequest{
		Key:   "key",
		Limit: ratelimit.Limit{Rate: 1, Period: 20 * time.Millisecond, Burst: 1},
		Cost:  1,
	})
	require.NoError(t, err)

	redisKey := store.prefix + "key"
	ttl, err := client.PTTL(t.Context(), redisKey).Result()
	require.NoError(t, err)
	assert.Positive(t, ttl)
	assert.LessOrEqual(t, ttl, 20*time.Millisecond)
	require.Eventually(t, func() bool {
		exists, existsErr := client.Exists(t.Context(), redisKey).Result()
		return existsErr == nil && exists == 0
	}, time.Second, time.Millisecond)
}

func TestStoreConcurrentTakeDoesNotExceedBurst(t *testing.T) {
	store, client := newTestStore(t)
	cleanupKeys(t, client, store, "key")
	const (
		callers = 32
		burst   = 5
	)
	request := ratelimit.TakeRequest{
		Key:   "key",
		Limit: ratelimit.Limit{Rate: 1, Period: time.Minute, Burst: burst},
		Cost:  1,
	}

	var allowed atomic.Int32
	var wg sync.WaitGroup
	errs := make(chan error, callers)
	wg.Add(callers)
	for range callers {
		go func() {
			defer wg.Done()
			decision, err := store.Take(t.Context(), request)
			if err != nil {
				errs <- err
				return
			}
			if decision.Allowed {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
	assert.Equal(t, int32(burst), allowed.Load())
}

func TestStoreContext(t *testing.T) {
	client := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})
	t.Cleanup(func() { _ = client.Close() })
	store := New(client)
	request := ratelimit.TakeRequest{
		Key:   "key",
		Limit: ratelimit.PerSecond(1),
		Cost:  1,
	}

	//nolint:staticcheck // Verifies the documented nil Context behavior.
	_, err := store.Take(nil, request)
	assert.ErrorIs(t, err, ratelimit.ErrInvalidContext)
	//nolint:staticcheck // Verifies the documented nil Context behavior.
	assert.ErrorIs(t, store.Reset(nil, request.Key), ratelimit.ErrInvalidContext)

	expected := errors.New("canceled")
	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(expected)
	_, err = store.Take(ctx, request)
	assert.ErrorIs(t, err, expected)
	assert.ErrorIs(t, store.Reset(ctx, request.Key), expected)
}

func TestStoreWrapsClientErrors(t *testing.T) {
	client := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})
	require.NoError(t, client.Close())
	store := New(client)
	request := ratelimit.TakeRequest{
		Key:   "key",
		Limit: ratelimit.PerSecond(1),
		Cost:  1,
	}

	_, err := store.Take(t.Context(), request)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `take "fries:ratelimit:key"`)
	err = store.Reset(t.Context(), request.Key)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `reset "fries:ratelimit:key"`)
}

func TestParseDecision(t *testing.T) {
	limit := ratelimit.PerSecond(5)
	decision, err := parseDecision(
		[]any{int64(0), int64(2), int64(3), int64(4)},
		limit,
	)
	require.NoError(t, err)
	assert.Equal(t, ratelimit.Decision{
		Limit:      limit,
		Allowed:    false,
		Remaining:  2,
		RetryAfter: 3 * time.Microsecond,
		ResetAfter: 4 * time.Microsecond,
	}, decision)

	tests := [][]any{
		nil,
		{int64(1)},
		{"1", int64(0), int64(0), int64(0)},
		{int64(2), int64(0), int64(0), int64(0)},
		{int64(1), int64(-1), int64(0), int64(0)},
		{int64(1), int64(6), int64(0), int64(0)},
		{int64(1), int64(0), int64(-1), int64(0)},
		{int64(1), int64(0), int64(0), int64(-1)},
		{int64(1), int64(0), int64(^uint64(0) >> 1), int64(0)},
		{int64(1), int64(0), int64(0), int64(^uint64(0) >> 1)},
	}
	for _, raw := range tests {
		_, err = parseDecision(raw, limit)
		assert.Error(t, err)
	}
}

func newTestStore(t *testing.T) (*Store, *goredis.Client) {
	t.Helper()
	if testing.Short() {
		t.Skip("Redis integration test")
	}

	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	client := goredis.NewClient(&goredis.Options{
		Addr:         addr,
		DialTimeout:  time.Second,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	})
	t.Cleanup(func() { _ = client.Close() })

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis is unavailable at %s: %v", addr, err)
	}

	prefix := fmt.Sprintf(
		"fries:test:ratelimit:%d:%d",
		time.Now().UnixNano(),
		testPrefixSequence.Add(1),
	)
	return New(client, WithPrefix(prefix)), client
}

func cleanupKeys(t *testing.T, client *goredis.Client, store *Store, keys ...string) {
	t.Helper()
	t.Cleanup(func() {
		redisKeys := make([]string, len(keys))
		for i, key := range keys {
			redisKeys[i] = store.prefix + key
		}
		_ = client.Del(context.WithoutCancel(t.Context()), redisKeys...).Err()
	})
}
