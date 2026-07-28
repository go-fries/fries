package redis

import (
	"testing"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	assert.Panics(t, func() {
		New(nil)
	})

	var client *goredis.Client
	assert.Panics(t, func() {
		New(client)
	})

	client = goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})
	t.Cleanup(func() { _ = client.Close() })
	store := New(client, nil, WithPrefix("billing:idempotency::"))
	assert.Equal(t, "billing:idempotency:", store.prefix)
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
			option: WithPrefix("billing:idempotency"),
			prefix: "billing:idempotency:",
		},
		"trailing colons": {
			option: WithPrefix("billing:idempotency::"),
			prefix: "billing:idempotency:",
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
