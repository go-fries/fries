package redis

import (
	"context"
	"testing"
	"time"

	"github.com/go-fries/fries/codec/json/v3"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

func createRedisClient(t *testing.T) redis.UniversalClient {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	require.NoError(t, client.Ping(ctx).Err())

	t.Cleanup(func() {
		client.FlushAll(ctx)
	})

	return client
}

func TestPositioner(t *testing.T) {
	positioner := NewPositioner(createRedisClient(t), WithPrefix("canal"), WithCodec(json.Codec))
	assert.Equal(t, "canal:"+name, positioner.prefix+name)
	assert.Equal(t, json.Codec, positioner.codec)

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
	positioner := NewBufferedPositioner(createRedisClient(t),
		WithPrefix("buffered:canal"), WithCodec(json.Codec),
		WithFlushInterval(5*time.Second), WithBatchSize(100),
	)
	assert.Equal(t, "buffered:canal:"+name, positioner.prefix+name)
	assert.Equal(t, json.Codec, positioner.codec)
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
	positioner := NewBufferedPositioner(createRedisClient(t),
		WithPrefix("buffered:batch"),
		WithBatchSize(3),
	)

	for i := 1; i <= 2; i++ {
		value := mysql.Position{
			Name: "mysql-bin.000001",
			Pos:  uint32(i * 1000),
		}
		assert.NoError(t, positioner.Set(ctx, value))
	}

	// Check that only the latest position is returned (buffered, not flushed)
	pos, err := positioner.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint32(2000), pos.Pos)

	// Add one more position to trigger batch flush
	final := mysql.Position{
		Name: "mysql-bin.000001",
		Pos:  3000,
	}
	assert.NoError(t, positioner.Set(ctx, final))

	// Check that position was flushed to Redis
	time.Sleep(100 * time.Millisecond) // Allow time for flush
	pos, err = positioner.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint32(3000), pos.Pos)
}

func TestBufferedPositioner_FlushInterval(t *testing.T) {
	positioner := NewBufferedPositioner(createRedisClient(t),
		WithPrefix("buffered:interval"),
		WithFlushInterval(500*time.Millisecond),
		WithBatchSize(100), // Large batch size to ensure interval triggers first
	)

	// Set position
	value := mysql.Position{
		Name: "mysql-bin.000001",
		Pos:  5000,
	}
	assert.NoError(t, positioner.Set(ctx, value))

	// Wait for flush interval
	time.Sleep(600 * time.Millisecond)

	// Check that position was flushed to Redis
	pos, err := positioner.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint32(5000), pos.Pos)
}
