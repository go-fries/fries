package redis

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedis_FlushWithoutPrefix(t *testing.T) {
	hook := &flushHook{keys: []string{"test:flush", "cache:redis:test:flush"}}
	client := redis.NewClient(&redis.Options{Addr: "unused:6379"})
	client.AddHook(hook)
	t.Cleanup(func() { assert.NoError(t, client.Close()) })

	ok, err := New(client).Flush(t.Context())
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, [][]any{
		{"scan", uint64(0), "match", "*", "count", int64(flushScanCount)},
		{"del", "test:flush", "cache:redis:test:flush"},
	}, hook.Commands())
}

func TestRedis_FlushClusterClient(t *testing.T) {
	hooks := map[string]*flushHook{
		"first:6379":  {keys: []string{"cache:test:first"}},
		"second:6379": {keys: []string{"cache:test:second"}},
	}
	client := redis.NewClusterClient(&redis.ClusterOptions{
		ClusterSlots: func(context.Context) ([]redis.ClusterSlot, error) {
			return []redis.ClusterSlot{
				{Start: 0, End: 8191, Nodes: []redis.ClusterNode{{Addr: "first:6379"}}},
				{Start: 8192, End: 16383, Nodes: []redis.ClusterNode{{Addr: "second:6379"}}},
			}, nil
		},
		NewClient: func(opts *redis.Options) *redis.Client {
			client := redis.NewClient(opts)
			client.AddHook(hooks[opts.Addr])
			return client
		},
	})
	t.Cleanup(func() { assert.NoError(t, client.Close()) })

	ok, err := New(client, Prefix("cache:test")).Flush(t.Context())
	require.NoError(t, err)
	assert.True(t, ok)
	for addr, hook := range hooks {
		assert.Equal(t, [][]any{
			{"scan", uint64(0), "match", "cache:test:*", "count", int64(flushScanCount)},
			{"del", hook.keys[0]},
		}, hook.Commands(), addr)
	}
}

// flushHook observes SCAN/DEL without sending commands to an external server.
type flushHook struct {
	mu       sync.Mutex
	keys     []string
	commands [][]any
}

func (h *flushHook) DialHook(redis.DialHook) redis.DialHook {
	return func(context.Context, string, string) (net.Conn, error) {
		return nil, fmt.Errorf("unexpected Redis dial")
	}
}

func (h *flushHook) ProcessHook(redis.ProcessHook) redis.ProcessHook {
	return func(_ context.Context, cmd redis.Cmder) error {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.commands = append(h.commands, append([]any(nil), cmd.Args()...))
		switch cmd := cmd.(type) {
		case *redis.ScanCmd:
			cmd.SetVal(h.keys, 0)
		case *redis.IntCmd:
			if cmd.Name() != "del" {
				return fmt.Errorf("unexpected Redis command: %s", cmd.Name())
			}
			cmd.SetVal(int64(len(cmd.Args()) - 1))
		default:
			return fmt.Errorf("unexpected Redis command: %s", cmd.Name())
		}
		return nil
	}
}

func (h *flushHook) ProcessPipelineHook(redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(context.Context, []redis.Cmder) error {
		return fmt.Errorf("unexpected Redis pipeline")
	}
}

func (h *flushHook) Commands() [][]any {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([][]any(nil), h.commands...)
}
