package redis

import (
	"context"
	"crypto/rand"
	"os"
	"testing"
	"time"

	"github.com/go-fries/fries/cache/v4"
	"github.com/go-fries/fries/codec/json/v4"
	"github.com/go-fries/fries/locker/v4"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const noExpiration = -1 * time.Nanosecond

func newTestStore(t *testing.T, keys ...string) *Store {
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

	store := New(client, Prefix("fries:test:cache:"+rand.Text()), Codec(json.Codec{}))
	redisKeys := make([]string, 0, 2*len(keys))
	for _, key := range keys {
		redisKeys = append(redisKeys, store.prefix+key, "locker:"+store.prefix+key)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 3*time.Second)
		defer cancel()
		if len(redisKeys) > 0 {
			assert.NoError(t, client.Del(ctx, redisKeys...).Err())
		}
	})
	return store
}

func TestRedis_Prefix(t *testing.T) {
	store := &Store{}
	Prefix("cache:test")(store)
	assert.Equal(t, "cache:test:", store.prefix)

	Prefix("cache:test:")(store)
	assert.Equal(t, "cache:test:", store.prefix)
}

func TestRedis_Base(t *testing.T) {
	store := newTestStore(t, "test")
	ctx := t.Context()

	ok, err := store.Put(ctx, "test", "test", time.Second)
	require.NoError(t, err)
	require.True(t, ok)

	var value string
	require.NoError(t, store.Get(ctx, "test", &value))
	assert.Equal(t, "test", value)

	ok, err = store.Has(ctx, "test")
	require.NoError(t, err)
	assert.True(t, ok)

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		ok, err := store.Has(ctx, "test")
		assert.NoError(c, err)
		assert.False(c, ok)
	}, 5*time.Second, 20*time.Millisecond)
}

func TestRedis_IncrAndDecr(t *testing.T) {
	store := newTestStore(t, "test:inc", "test:inc:type")
	ctx := t.Context()

	ok, err := store.Forget(ctx, "test:inc")
	require.NoError(t, err)
	assert.False(t, ok)

	value, err := store.Increment(ctx, "test:inc", 1)
	require.NoError(t, err)
	assert.Equal(t, 1, value)

	value, err = store.Increment(ctx, "test:inc", 10)
	require.NoError(t, err)
	assert.Equal(t, 11, value)

	value, err = store.Decrement(ctx, "test:inc", 1)
	require.NoError(t, err)
	assert.Equal(t, 10, value)

	ok, err = store.Put(ctx, "test:inc:type", "test", time.Minute)
	require.NoError(t, err)
	require.True(t, ok)

	value, err = store.Increment(ctx, "test:inc:type", 1)
	assert.Error(t, err)
	assert.Zero(t, value)

	value, err = store.Decrement(ctx, "test:inc:type", 1)
	assert.Error(t, err)
	assert.Zero(t, value)
}

func TestRedis_Forever(t *testing.T) {
	store := newTestStore(t, "test:forever", "test:forever:ttl")
	ctx := t.Context()

	ok, err := store.Forever(ctx, "test:forever", "test")
	require.NoError(t, err)
	require.True(t, ok)

	ttl, err := store.redis.TTL(ctx, store.prefix+"test:forever").Result()
	require.NoError(t, err)
	assert.Equal(t, noExpiration, ttl)

	ok, err = store.Put(ctx, "test:forever:ttl", "test", time.Minute)
	require.NoError(t, err)
	require.True(t, ok)

	ttl, err = store.redis.TTL(ctx, store.prefix+"test:forever:ttl").Result()
	require.NoError(t, err)
	assert.Positive(t, ttl)

	ok, err = store.Forever(ctx, "test:forever:ttl", "forever-value")
	require.NoError(t, err)
	require.True(t, ok)

	var value string
	require.NoError(t, store.Get(ctx, "test:forever:ttl", &value))
	assert.Equal(t, "forever-value", value)

	ttl, err = store.redis.TTL(ctx, store.prefix+"test:forever:ttl").Result()
	require.NoError(t, err)
	assert.Equal(t, noExpiration, ttl)
}

func TestRedis_Flush(t *testing.T) {
	store := newTestStore(t, "test:flush", "test:flush:another")
	otherStore := newTestStore(t, "test:flush")
	ctx := t.Context()

	for _, key := range []string{"test:flush", "test:flush:another"} {
		ok, err := store.Forever(ctx, key, "test")
		require.NoError(t, err)
		require.True(t, ok)
	}
	ok, err := otherStore.Forever(ctx, "test:flush", "other-value")
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = store.Flush(ctx)
	require.NoError(t, err)
	assert.True(t, ok)

	for _, key := range []string{"test:flush", "test:flush:another"} {
		hasKey, err := store.Has(ctx, key)
		require.NoError(t, err)
		assert.False(t, hasKey)
	}
	var value string
	require.NoError(t, otherStore.Get(ctx, "test:flush", &value))
	assert.Equal(t, "other-value", value)
}

func TestRedis_Add(t *testing.T) {
	store := newTestStore(t, "test:add")
	ctx := t.Context()

	ok, err := store.Add(ctx, "test:add", "first", time.Second)
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = store.Add(ctx, "test:add", "replacement", time.Minute)
	require.NoError(t, err)
	assert.False(t, ok)

	var value string
	require.NoError(t, store.Get(ctx, "test:add", &value))
	assert.Equal(t, "first", value)

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		exists, err := store.Has(ctx, "test:add")
		assert.NoError(c, err)
		assert.False(c, exists)
	}, 5*time.Second, 20*time.Millisecond)

	ok, err = store.Add(ctx, "test:add", "replacement", time.Minute)
	require.NoError(t, err)
	assert.True(t, ok)
	require.NoError(t, store.Get(ctx, "test:add", &value))
	assert.Equal(t, "replacement", value)
}

func TestRedis_Lock(t *testing.T) {
	store := newTestStore(t, "test")
	ctx := t.Context()
	lock := store.Lock("test", 5*time.Second)
	lease, err := lock.TryAcquire(ctx)
	require.NoError(t, err)
	require.NotNil(t, lease)
	exists, err := store.redis.Exists(ctx, "locker:"+store.prefix+"test").Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), exists)

	err = locker.Try(ctx, store.Lock("test", 5*time.Second), func(context.Context) error {
		return nil
	})
	assert.ErrorIs(t, err, locker.ErrNotAcquired)
	require.NoError(t, lease.Release(ctx))

	err = locker.Try(ctx, store.Lock("test", 5*time.Second), func(context.Context) error {
		return nil
	})
	assert.NoError(t, err)
}

func TestRedis_ErrNotFound(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()

	ok, err := store.Has(ctx, "test:notfound:has")
	require.NoError(t, err)
	assert.False(t, ok)

	var value string
	err = store.Get(ctx, "test:notfound:get", &value)
	assert.ErrorIs(t, err, cache.ErrNotFound)
	assert.Empty(t, value)
}
