package redis

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-fries/fries/idempotency/v4"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreLifecycle(t *testing.T) {
	store, client := newTestStore(t)
	ctx := t.Context()
	cleanupKeys(t, client, store, "key")

	begin, err := store.Begin(ctx, idempotency.BeginRequest{
		Key:         "key",
		Token:       "token-1",
		Fingerprint: "request",
		TTL:         time.Minute,
	})
	require.NoError(t, err)
	assert.Equal(t, idempotency.BeginAcquired, begin.Status)

	begin, err = store.Begin(ctx, idempotency.BeginRequest{
		Key:         "key",
		Token:       "token-2",
		Fingerprint: "request",
		TTL:         time.Minute,
	})
	require.NoError(t, err)
	assert.Equal(t, idempotency.BeginInProgress, begin.Status)

	data := []byte{0, 1, 2, 255}
	require.NoError(t, store.Complete(ctx, idempotency.CompleteRequest{
		Key:    "key",
		Token:  "token-1",
		Result: data,
		TTL:    time.Hour,
	}))
	begin, err = store.Begin(ctx, idempotency.BeginRequest{
		Key:         "key",
		Token:       "token-3",
		Fingerprint: "request",
		TTL:         time.Minute,
	})
	require.NoError(t, err)
	assert.Equal(t, idempotency.BeginCompleted, begin.Status)
	assert.Equal(t, data, begin.Result)
}

func TestStoreFingerprintAndAbort(t *testing.T) {
	store, client := newTestStore(t)
	ctx := t.Context()
	cleanupKeys(t, client, store, "key")

	_, err := store.Begin(ctx, idempotency.BeginRequest{
		Key:         "key",
		Token:       "token",
		Fingerprint: "first",
		TTL:         time.Minute,
	})
	require.NoError(t, err)
	_, err = store.Begin(ctx, idempotency.BeginRequest{
		Key:         "key",
		Token:       "other",
		Fingerprint: "second",
		TTL:         time.Minute,
	})
	assert.ErrorIs(t, err, idempotency.ErrKeyConflict)

	require.NoError(t, store.Abort(ctx, idempotency.AbortRequest{
		Key:   "key",
		Token: "token",
	}))
	begin, err := store.Begin(ctx, idempotency.BeginRequest{
		Key:   "key",
		Token: "next",
		TTL:   time.Minute,
	})
	require.NoError(t, err)
	assert.Equal(t, idempotency.BeginAcquired, begin.Status)
}

func TestStoreRejectsLostClaims(t *testing.T) {
	store, client := newTestStore(t)
	ctx := t.Context()
	cleanupKeys(t, client, store, "key")
	_, err := store.Begin(ctx, idempotency.BeginRequest{
		Key:   "key",
		Token: "owner",
		TTL:   time.Minute,
	})
	require.NoError(t, err)

	assert.ErrorIs(t, store.Complete(ctx, idempotency.CompleteRequest{
		Key:   "key",
		Token: "other",
		TTL:   time.Hour,
	}), idempotency.ErrClaimLost)
	assert.ErrorIs(t, store.Abort(ctx, idempotency.AbortRequest{
		Key:   "key",
		Token: "other",
	}), idempotency.ErrClaimLost)
}

func TestStoreClaimExpires(t *testing.T) {
	store, client := newTestStore(t)
	ctx := t.Context()
	cleanupKeys(t, client, store, "key")
	_, err := store.Begin(ctx, idempotency.BeginRequest{
		Key:   "key",
		Token: "old",
		TTL:   time.Millisecond,
	})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		begin, beginErr := store.Begin(ctx, idempotency.BeginRequest{
			Key:   "key",
			Token: "new",
			TTL:   time.Minute,
		})
		return beginErr == nil && begin.Status == idempotency.BeginAcquired
	}, time.Second, time.Millisecond)

	assert.ErrorIs(t, store.Complete(ctx, idempotency.CompleteRequest{
		Key:   "key",
		Token: "old",
		TTL:   time.Hour,
	}), idempotency.ErrClaimLost)
}

func TestStoreCompletedRecordExpires(t *testing.T) {
	store, client := newTestStore(t)
	ctx := t.Context()
	cleanupKeys(t, client, store, "key")
	_, err := store.Begin(ctx, idempotency.BeginRequest{
		Key:   "key",
		Token: "owner",
		TTL:   time.Minute,
	})
	require.NoError(t, err)
	require.NoError(t, store.Complete(ctx, idempotency.CompleteRequest{
		Key:   "key",
		Token: "owner",
		TTL:   time.Millisecond,
	}))

	require.Eventually(t, func() bool {
		begin, beginErr := store.Begin(ctx, idempotency.BeginRequest{
			Key:   "key",
			Token: "next",
			TTL:   time.Minute,
		})
		return beginErr == nil && begin.Status == idempotency.BeginAcquired
	}, time.Second, time.Millisecond)
}

func TestStoreOnlyGrantsOneConcurrentClaim(t *testing.T) {
	store, client := newTestStore(t)
	ctx := t.Context()
	cleanupKeys(t, client, store, "key")
	const callers = 32
	var acquired atomic.Int32
	var inProgress atomic.Int32
	errs := make(chan error, callers)
	var wg sync.WaitGroup
	wg.Add(callers)

	for i := range callers {
		go func() {
			defer wg.Done()
			result, err := store.Begin(ctx, idempotency.BeginRequest{
				Key:   "key",
				Token: fmt.Sprintf("token-%d", i),
				TTL:   time.Minute,
			})
			if err != nil {
				errs <- err
				return
			}
			switch result.Status {
			case idempotency.BeginAcquired:
				acquired.Add(1)
			case idempotency.BeginInProgress:
				inProgress.Add(1)
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
	assert.Equal(t, int32(1), acquired.Load())
	assert.Equal(t, int32(callers-1), inProgress.Load())
}

func TestStorePrefixesAreIsolated(t *testing.T) {
	first, client := newTestStore(t)
	second := New(client, WithPrefix(first.prefix+"other"))
	ctx := t.Context()
	cleanupKeys(t, client, first, "key")
	cleanupKeys(t, client, second, "key")

	for _, store := range []*Store{first, second} {
		result, err := store.Begin(ctx, idempotency.BeginRequest{
			Key:   "key",
			Token: "token",
			TTL:   time.Minute,
		})
		require.NoError(t, err)
		assert.Equal(t, idempotency.BeginAcquired, result.Status)
	}
}

func TestStoreContext(t *testing.T) {
	client := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})
	t.Cleanup(func() { _ = client.Close() })
	store := New(client)
	request := idempotency.BeginRequest{
		Key:   "key",
		Token: "token",
		TTL:   time.Minute,
	}

	//nolint:staticcheck // Verifies the documented nil Context behavior.
	_, err := store.Begin(nil, request)
	assert.ErrorIs(t, err, idempotency.ErrInvalidContext)

	expected := errors.New("canceled")
	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(expected)
	_, err = store.Begin(ctx, request)
	assert.ErrorIs(t, err, expected)
	assert.ErrorIs(t, store.Complete(ctx, idempotency.CompleteRequest{}), expected)
	assert.ErrorIs(t, store.Abort(ctx, idempotency.AbortRequest{}), expected)
}

func TestMilliseconds(t *testing.T) {
	assert.Equal(t, int64(1), milliseconds(time.Nanosecond))
	assert.Equal(t, int64(1), milliseconds(time.Millisecond))
	assert.Equal(t, int64(1500), milliseconds(1500*time.Millisecond))
}

func newTestStore(t *testing.T) (*Store, *goredis.Client) {
	t.Helper()
	if testing.Short() {
		t.Skip("Redis integration test")
	}

	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	client := goredis.NewClient(&goredis.Options{
		Addr:                  addr,
		DialTimeout:           time.Second,
		ReadTimeout:           time.Second,
		WriteTimeout:          time.Second,
		ContextTimeoutEnabled: true,
		MaxRetries:            -1,
	})
	t.Cleanup(func() { assert.NoError(t, client.Close()) })

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	require.NoError(t, client.Ping(ctx).Err(), "Redis is unavailable at %s", addr)

	prefix := "fries:test:idempotency:" + rand.Text()
	return New(client, WithPrefix(prefix)), client
}

func cleanupKeys(t *testing.T, client *goredis.Client, store *Store, keys ...string) {
	t.Helper()
	t.Cleanup(func() {
		redisKeys := make([]string, len(keys))
		for i, key := range keys {
			redisKeys[i] = store.prefix + key
		}
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 3*time.Second)
		defer cancel()
		assert.NoError(t, client.Del(ctx, redisKeys...).Err())
	})
}
