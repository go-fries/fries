package redispositioner

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
