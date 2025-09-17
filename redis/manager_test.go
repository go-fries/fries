package redis

import (
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestManager(t *testing.T) {
	ctx := t.Context()
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	m := New(rdb)

	m.Register("rdb2", rdb)

	assert.Equal(t, rdb, m.UniversalClient)
	assert.Equal(t, rdb, m.Conn())
	assert.Equal(t, rdb, m.Conn("rdb2"))

	assert.Panics(t, func() {
		m.Conn("rdb3")
	})

	var (
		key1 = "redis:manager:key1"
		key2 = "redis:manager:key2"
		key3 = "redis:manager:key3"
	)

	m.Set(ctx, key1, "value1", time.Second*10)
	m.Conn().Set(ctx, key2, "value2", time.Second*10)
	m.Conn("rdb2").Set(ctx, key3, "value3", time.Second*10)

	// default connection
	val, err := m.Get(ctx, key1).Result()
	assert.NoError(t, err)
	assert.Equal(t, "value1", val)

	// empty name connection
	val, err = m.Conn().Get(ctx, key2).Result()
	assert.NoError(t, err)
	assert.Equal(t, "value2", val)

	// use named connection
	val, err = m.Conn("rdb2").Get(ctx, key3).Result()
	assert.NoError(t, err)
	assert.Equal(t, "value3", val)
}
