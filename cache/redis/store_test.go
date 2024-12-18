package redis

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"github.com/go-kratos-ecosystem/components/v2/cache"
	"github.com/go-kratos-ecosystem/components/v2/codec/json"
	"github.com/go-kratos-ecosystem/components/v2/locker"
)

var ctx = context.Background()

func createRedis(t *testing.T) redis.UniversalClient {
	client := redis.NewClient(&redis.Options{
		Addr: ":6379",
	})
	t.Cleanup(func() {
		client.FlushAll(ctx)
	})
	return client
}

func TestRedis_Prefix(t *testing.T) {
	opt := &options{}
	Prefix("cache:test")(opt)
	assert.Equal(t, "cache:test:", opt.prefix)

	Prefix("cache:test:")(opt)
	assert.Equal(t, "cache:test:", opt.prefix)
}

func TestRedis_Base(t *testing.T) {
	store := New(createRedis(t), Prefix("cache:redis"), Codec(json.Codec))

	ok1, err := store.Put(ctx, "test", "test", time.Second)
	assert.Nil(t, err)
	assert.True(t, ok1)

	var v string
	assert.Nil(t, store.Get(ctx, "test", &v))
	assert.Equal(t, "test", v)

	ok2, err := store.Has(ctx, "test")
	assert.Nil(t, err)
	assert.True(t, ok2)

	time.Sleep(time.Second + 500*time.Millisecond)

	ok3, err := store.Has(ctx, "test")
	assert.Nil(t, err)
	assert.False(t, ok3)
}

func TestRedis_IncrAndDecr(t *testing.T) {
	store := New(createRedis(t), Prefix("cache:redis"))

	_, err := store.Forget(ctx, "test:inc")
	assert.Nil(t, err)

	v1, err := store.Increment(ctx, "test:inc", 1)
	assert.Nil(t, err)
	assert.Equal(t, 1, v1)

	v2, err := store.Increment(ctx, "test:inc", 10)
	assert.Nil(t, err)
	assert.Equal(t, 11, v2)

	v3, err := store.Decrement(ctx, "test:inc", 1)
	assert.Nil(t, err)
	assert.Equal(t, 10, v3)

	// put another type
	ok1, err := store.Put(ctx, "test:inc:type", "test", time.Second*3)
	assert.Nil(t, err)
	assert.True(t, ok1)

	v4, err := store.Increment(ctx, "test:inc:type", 1)
	t.Log(err)
	assert.Error(t, err)
	assert.Zero(t, v4)

	v5, err := store.Decrement(ctx, "test:inc:type", 1)
	assert.Error(t, err)
	assert.Zero(t, v5)
}

func TestRedis_Forever(t *testing.T) {
	client := createRedis(t)
	store := New(createRedis(t), Prefix("cache:redis"))

	ok1, err := store.Forever(ctx, "test:forever", "test")
	assert.Nil(t, err)
	assert.True(t, ok1)

	// ttl
	ttl, err := client.TTL(ctx, "cache:redis:test:forever").Result()
	assert.Nil(t, err)
	assert.Equal(t, time.Duration(redis.KeepTTL), ttl)
}

func TestRedis_Flush(t *testing.T) {
	store := New(createRedis(t), Prefix("cache:redis"))

	ok1, err := store.Put(ctx, "test:flush", "test", time.Second)
	assert.Nil(t, err)
	assert.True(t, ok1)

	ok2, err := store.Flush(ctx)
	assert.Nil(t, err)
	assert.True(t, ok2)

	ok3, err := store.Has(ctx, "test:flush")
	assert.NoError(t, err)
	assert.False(t, ok3)
}

func TestRedis_Add(t *testing.T) {
	store := New(createRedis(t), Prefix("cache:redis"))

	ok1, err := store.Add(ctx, "test:add", "test", time.Second)
	assert.Nil(t, err)
	assert.True(t, ok1)

	ok2, err := store.Add(ctx, "test:add", "test", time.Second)
	assert.Nil(t, err)
	assert.False(t, ok2)

	time.Sleep(time.Second + 500*time.Millisecond)
	ok3, err := store.Add(ctx, "test:add", "test", time.Second)
	assert.Nil(t, err)
	assert.True(t, ok3)
}

func TestRedis_Lock(t *testing.T) {
	r := New(createRedis(t))
	var wg sync.WaitGroup
	var s int64

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := r.Lock("test", 5*time.Second).Try(context.Background(), func() {
				time.Sleep(time.Second)
			})
			if err != nil {
				assert.True(t, errors.Is(err, locker.ErrLocked))
			} else {
				atomic.AddInt64(&s, 1)
			}
		}()
	}
	wg.Wait()
	assert.True(t, s > 0)
}

func TestRedis_ErrNotFound(t *testing.T) {
	store := New(createRedis(t), Prefix("cache:redis:notfound"))

	// Has
	ok1, err := store.Has(ctx, "test:notfound:has")
	assert.Nil(t, err)
	assert.False(t, ok1)

	// Get
	var v string
	err = store.Get(ctx, "test:notfound:get", &v)
	assert.True(t, errors.Is(err, cache.ErrNotFound))
	assert.Empty(t, v)
}
