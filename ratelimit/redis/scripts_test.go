package redis

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
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
	if testing.Short() {
		b.Skip("Redis integration benchmark")
	}

	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	client := goredis.NewClient(&goredis.Options{
		Addr:         addr,
		DialTimeout:  time.Second,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	})
	b.Cleanup(func() { _ = client.Close() })

	ctx, cancel := context.WithTimeout(b.Context(), time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		b.Skipf("Redis is unavailable at %s: %v", addr, err)
	}

	prefix := fmt.Sprintf(
		"fries:benchmark:ratelimit:%d:%d",
		time.Now().UnixNano(),
		testPrefixSequence.Add(1),
	)
	return New(client, WithPrefix(prefix)), client
}

func cleanupBenchmarkKey(
	b *testing.B,
	client *goredis.Client,
	key string,
) {
	b.Helper()
	b.Cleanup(func() {
		_ = client.Del(context.WithoutCancel(b.Context()), key).Err()
	})
}
