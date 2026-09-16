package redis

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-fries/fries/codec/json/v4"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

func createRedisClient(t *testing.T) redis.UniversalClient {
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

	t.Cleanup(func() {
		assert.NoError(t, client.Close())
	})
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	require.NoError(t, client.Ping(ctx).Err(), "Redis is unavailable at %s", addr)

	return client
}

func TestPositioner(t *testing.T) {
	client := createRedisClient(t)
	prefix := "fries:test:positioner:" + rand.Text()
	positioner := NewPositioner(client, WithPrefix(prefix), WithCodec(json.Codec{}))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 3*time.Second)
		defer cancel()
		assert.NoError(t, client.Del(ctx, positioner.prefix+name).Err())
	})
	assert.Equal(t, prefix+":"+name, positioner.prefix+name)
	assert.Equal(t, json.Codec{}, positioner.codec)

	pos, err := positioner.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, zeroPosition, pos)

	value := mysql.Position{
		Name: "mysql-bin.000001",
		Pos:  123456,
	}

	assert.NoError(t, positioner.Set(ctx, value))

	pos, err = positioner.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, value, pos)
}

func TestBufferedPositioner(t *testing.T) {
	client := createRedisClient(t)
	prefix := "fries:test:positioner:" + rand.Text()
	positioner := NewBufferedPositioner(
		client,
		WithPrefix(prefix), WithCodec(json.Codec{}),
		WithFlushInterval(5*time.Second), WithBatchSize(100),
	)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 3*time.Second)
		defer cancel()
		positioner.Close(ctx)
		assert.NoError(t, client.Del(ctx, positioner.prefix+name).Err())
	})
	assert.Equal(t, prefix+":"+name, positioner.prefix+name)
	assert.Equal(t, json.Codec{}, positioner.codec)
	assert.Equal(t, 5*time.Second, positioner.flushInterval)
	assert.Equal(t, 100, positioner.batchSize)

	pos, err := positioner.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, zeroPosition, pos)

	value := mysql.Position{
		Name: "mysql-bin.000001",
		Pos:  123456,
	}

	assert.NoError(t, positioner.Set(ctx, value))

	pos, err = positioner.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, value, pos)
}

func TestBufferedPositioner_Batching(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := newPositionClient()
		positioner := NewBufferedPositioner(client, WithPrefix("batch"), WithBatchSize(3), WithFlushInterval(time.Hour))
		defer func() {
			synctest.Wait()
			positioner.Close(t.Context())
			synctest.Wait()
		}()
		baseline := mysql.Position{Name: "mysql-bin.000001", Pos: 1000}
		require.NoError(t, positioner.Set(t.Context(), baseline))
		assert.Equal(t, 1, client.Writes())
		for _, offset := range []uint32{1100, 1200} {
			require.NoError(t, positioner.Set(t.Context(), mysql.Position{Name: baseline.Name, Pos: offset}))
			stored, err := positioner.Positioner.Get(t.Context())
			require.NoError(t, err)
			assert.Equal(t, baseline, stored)
			assert.Equal(t, 1, client.Writes())
		}
		cached, err := positioner.Get(t.Context())
		require.NoError(t, err)
		assert.Equal(t, uint32(1200), cached.Pos)

		final := mysql.Position{Name: baseline.Name, Pos: 1300}
		require.NoError(t, positioner.Set(t.Context(), final))
		stored, err := positioner.Positioner.Get(t.Context())
		require.NoError(t, err)
		assert.Equal(t, final, stored)
		assert.Equal(t, 2, client.Writes())
	})
}

func TestBufferedPositioner_FlushInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const interval = time.Second
		client := newPositionClient()
		positioner := NewBufferedPositioner(client, WithPrefix("interval"), WithFlushInterval(interval), WithBatchSize(100))
		defer func() {
			// Wait for any synchronous fake-backed flush to finish before Close
			// stops the timer, so its callback cannot re-arm it afterward.
			synctest.Wait()
			positioner.Close(t.Context())
			synctest.Wait()
		}()
		baseline := mysql.Position{Name: "mysql-bin.000001", Pos: 1000}
		require.NoError(t, positioner.Set(t.Context(), baseline))
		assert.Equal(t, 1, client.Writes())

		for i := range 2 {
			value := mysql.Position{Name: baseline.Name, Pos: baseline.Pos + 100}
			require.NoError(t, positioner.Set(t.Context(), value))
			time.Sleep(interval - time.Nanosecond)
			synctest.Wait()
			stored, err := positioner.Positioner.Get(t.Context())
			require.NoError(t, err)
			assert.Equal(t, baseline, stored)
			assert.Equal(t, i+1, client.Writes())
			time.Sleep(time.Nanosecond)
			synctest.Wait()
			stored, err = positioner.Positioner.Get(t.Context())
			require.NoError(t, err)
			assert.Equal(t, value, stored)
			assert.Equal(t, i+2, client.Writes())
			baseline = value
		}
		positioner.Close(t.Context())
		synctest.Wait()
		assert.False(t, positioner.timer.Stop(), "Close must stop the re-armed timer")
		time.Sleep(2 * interval)
		synctest.Wait()
		assert.Equal(t, 3, client.Writes())
	})
}

// positionClient implements only the Redis operations used by Positioner.
// Its storage is independent of BufferedPositioner's in-memory cache.
type positionClient struct {
	redis.UniversalClient
	mu     sync.Mutex
	values map[string]string
	writes int
}

func newPositionClient() *positionClient {
	return &positionClient{values: make(map[string]string)}
}

func (c *positionClient) Get(ctx context.Context, key string) *redis.StringCmd {
	if err := ctx.Err(); err != nil {
		return redis.NewStringResult("", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	value, ok := c.values[key]
	if !ok {
		return redis.NewStringResult("", redis.Nil)
	}
	return redis.NewStringResult(value, nil)
}

func (c *positionClient) Set(ctx context.Context, key string, value any, _ time.Duration) *redis.StatusCmd {
	if err := ctx.Err(); err != nil {
		return redis.NewStatusResult("", err)
	}
	data, ok := value.([]byte)
	if !ok {
		return redis.NewStatusResult("", fmt.Errorf("unexpected encoded position type %T", value))
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = string(data)
	c.writes++
	return redis.NewStatusResult("OK", nil)
}

func (c *positionClient) Writes() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.writes
}
