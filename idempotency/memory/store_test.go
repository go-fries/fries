package memory

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-fries/fries/idempotency/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreLifecycle(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	store := New()
	store.now = func() time.Time {
		return now
	}
	ctx := t.Context()

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

	result := []byte{0, 1, 2, 255}
	require.NoError(t, store.Complete(ctx, idempotency.CompleteRequest{
		Key:    "key",
		Token:  "token-1",
		Result: result,
		TTL:    time.Hour,
	}))
	result[0] = 9

	begin, err = store.Begin(ctx, idempotency.BeginRequest{
		Key:         "key",
		Token:       "token-3",
		Fingerprint: "request",
		TTL:         time.Minute,
	})
	require.NoError(t, err)
	assert.Equal(t, idempotency.BeginCompleted, begin.Status)
	assert.Equal(t, []byte{0, 1, 2, 255}, begin.Result)
	begin.Result[0] = 8

	replayed, err := store.Begin(ctx, idempotency.BeginRequest{
		Key:         "key",
		Token:       "token-4",
		Fingerprint: "request",
		TTL:         time.Minute,
	})
	require.NoError(t, err)
	assert.Equal(t, []byte{0, 1, 2, 255}, replayed.Result)

	now = now.Add(time.Hour)
	begin, err = store.Begin(ctx, idempotency.BeginRequest{
		Key:         "key",
		Token:       "token-5",
		Fingerprint: "new-request",
		TTL:         time.Minute,
	})
	require.NoError(t, err)
	assert.Equal(t, idempotency.BeginAcquired, begin.Status)
}

func TestStoreFingerprintConflict(t *testing.T) {
	store := New()
	ctx := t.Context()
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

	begin, err := store.Begin(ctx, idempotency.BeginRequest{
		Key:   "key",
		Token: "without-fingerprint",
		TTL:   time.Minute,
	})
	require.NoError(t, err)
	assert.Equal(t, idempotency.BeginInProgress, begin.Status)
}

func TestStoreAbortAllowsRetry(t *testing.T) {
	store := New()
	ctx := t.Context()
	_, err := store.Begin(ctx, idempotency.BeginRequest{
		Key:   "key",
		Token: "token",
		TTL:   time.Minute,
	})
	require.NoError(t, err)
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
	store := New()
	ctx := t.Context()
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

func TestStoreExpiredOwnerCannotReplaceNewClaim(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	store := New()
	store.now = func() time.Time {
		return now
	}
	ctx := t.Context()
	_, err := store.Begin(ctx, idempotency.BeginRequest{
		Key:   "key",
		Token: "old",
		TTL:   time.Minute,
	})
	require.NoError(t, err)

	now = now.Add(time.Minute)
	begin, err := store.Begin(ctx, idempotency.BeginRequest{
		Key:   "key",
		Token: "new",
		TTL:   time.Minute,
	})
	require.NoError(t, err)
	assert.Equal(t, idempotency.BeginAcquired, begin.Status)

	assert.ErrorIs(t, store.Complete(ctx, idempotency.CompleteRequest{
		Key:   "key",
		Token: "old",
		TTL:   time.Hour,
	}), idempotency.ErrClaimLost)
	assert.ErrorIs(t, store.Abort(ctx, idempotency.AbortRequest{
		Key:   "key",
		Token: "old",
	}), idempotency.ErrClaimLost)
}

func TestStoreOnlyGrantsOneConcurrentClaim(t *testing.T) {
	store := New()
	ctx := t.Context()
	const callers = 32
	var acquired atomic.Int32
	var inProgress atomic.Int32
	var wg sync.WaitGroup
	wg.Add(callers)

	for i := range callers {
		go func() {
			defer wg.Done()
			result, err := store.Begin(ctx, idempotency.BeginRequest{
				Key:   "key",
				Token: string(rune(i + 1)),
				TTL:   time.Minute,
			})
			require.NoError(t, err)
			switch result.Status {
			case idempotency.BeginAcquired:
				acquired.Add(1)
			case idempotency.BeginInProgress:
				inProgress.Add(1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), acquired.Load())
	assert.Equal(t, int32(callers-1), inProgress.Load())
}

func TestStoreContext(t *testing.T) {
	store := New()
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
