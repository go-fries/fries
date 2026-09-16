package redis

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func BenchmarkTakeScript(b *testing.B) {
	store, client := newBenchmarkStore(b)

	b.Run("allowed", func(b *testing.B) {
		key := store.prefix + "allowed"
		cleanupBenchmarkKey(b, client, key)
		benchmarkTakeScript(b, store, key, 1, 1_000_000, 1)
	})

	b.Run("rejected", func(b *testing.B) {
		key := store.prefix + "rejected"
		cleanupBenchmarkKey(b, client, key)
		ctx := b.Context()
		_, err := takeScript.Run(
			ctx,
			store.client,
			[]string{key},
			int64(time.Minute/time.Microsecond),
			1,
			1,
		).Slice()
		if err != nil {
			b.Fatal(err)
		}

		benchmarkTakeScript(
			b,
			store,
			key,
			int64(time.Minute/time.Microsecond),
			1,
			1,
		)
	})

	b.Run("parallel", func(b *testing.B) {
		key := store.prefix + "parallel"
		cleanupBenchmarkKey(b, client, key)
		ctx := b.Context()
		keys := []string{key}

		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				if _, err := takeScript.Run(
					ctx,
					store.client,
					keys,
					1,
					1_000_000,
					1,
				).Slice(); err != nil {
					b.Error(err)
					return
				}
			}
		})
	})
}

func benchmarkTakeScript(
	b *testing.B,
	store *Store,
	key string,
	interval int64,
	burst int,
	cost int,
) {
	b.Helper()
	ctx := b.Context()
	keys := []string{key}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := takeScript.Run(
			ctx,
			store.client,
			keys,
			interval,
			burst,
			cost,
		).Slice(); err != nil {
			b.Fatal(err)
		}
	}
}

func newBenchmarkStore(b *testing.B) (*Store, *goredis.Client) {
	b.Helper()
	client := newRedisClient(b)
	prefix := "fries:benchmark:ratelimit:" + rand.Text()
	return New(client, WithPrefix(prefix)), client
}

func cleanupBenchmarkKey(
	b *testing.B,
	client *goredis.Client,
	key string,
) {
	b.Helper()
	b.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(b.Context()), 3*time.Second)
		defer cancel()
		assert.NoError(b, client.Del(ctx, key).Err())
	})
}
