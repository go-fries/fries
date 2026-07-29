package redis

import (
	"testing"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	assert.Equal(t, defaultPrefix, newConfig().prefix)
	assert.Equal(t, defaultPrefix, newConfig(nil).prefix)
	assert.Equal(t, "app:", newConfig(WithPrefix("app:::")).prefix)
	assert.Equal(t, defaultPrefix, newConfig(WithPrefix("")).prefix)
	assert.Equal(t, defaultPrefix, newConfig(WithPrefix(":::")).prefix)
}

func TestNew(t *testing.T) {
	client := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})
	t.Cleanup(func() { _ = client.Close() })

	store := New(client, WithPrefix("app"))
	assert.Same(t, client, store.client)
	assert.Equal(t, "app:", store.prefix)
	assert.PanicsWithValue(t, "ratelimit/redis: nil client", func() {
		New(nil)
	})
}
