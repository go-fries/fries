package memory

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-fries/fries/ratelimit/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreBurstAndRecovery(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	store := newStoreAt(&now)
	request := ratelimit.TakeRequest{
		Key:   "user:1",
		Limit: ratelimit.Limit{Rate: 1, Period: time.Second, Burst: 3},
		Cost:  1,
	}

	for remaining := 2; remaining >= 0; remaining-- {
		decision, err := store.Take(t.Context(), request)
		require.NoError(t, err)
		assert.True(t, decision.Allowed)
		assert.Equal(t, remaining, decision.Remaining)
		assert.Equal(t, time.Duration(3-remaining)*time.Second, decision.ResetAfter)
		assert.Zero(t, decision.RetryAfter)
	}

	decision, err := store.Take(t.Context(), request)
	require.NoError(t, err)
	assert.False(t, decision.Allowed)
	assert.Zero(t, decision.Remaining)
	assert.Equal(t, time.Second, decision.RetryAfter)
	assert.Equal(t, 3*time.Second, decision.ResetAfter)

	now = now.Add(time.Second)
	decision, err = store.Take(t.Context(), request)
	require.NoError(t, err)
	assert.True(t, decision.Allowed)
	assert.Zero(t, decision.Remaining)
	assert.Equal(t, 3*time.Second, decision.ResetAfter)
}

func TestStoreAllowNIsAllOrNothing(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	store := newStoreAt(&now)
	limit := ratelimit.Limit{Rate: 1, Period: time.Second, Burst: 4}

	decision, err := store.Take(t.Context(), ratelimit.TakeRequest{
		Key:   "key",
		Limit: limit,
		Cost:  3,
	})
	require.NoError(t, err)
	assert.True(t, decision.Allowed)
	assert.Equal(t, 1, decision.Remaining)

	decision, err = store.Take(t.Context(), ratelimit.TakeRequest{
		Key:   "key",
		Limit: limit,
		Cost:  2,
	})
	require.NoError(t, err)
	assert.False(t, decision.Allowed)
	assert.Equal(t, 1, decision.Remaining)
	assert.Equal(t, time.Second, decision.RetryAfter)

	decision, err = store.Take(t.Context(), ratelimit.TakeRequest{
		Key:   "key",
		Limit: limit,
		Cost:  1,
	})
	require.NoError(t, err)
	assert.True(t, decision.Allowed)
	assert.Zero(t, decision.Remaining)
}

func TestStoreKeysAreIndependent(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	store := newStoreAt(&now)
	limit := ratelimit.Limit{Rate: 1, Period: time.Minute, Burst: 1}

	for _, key := range []string{"first", "second"} {
		decision, err := store.Take(t.Context(), ratelimit.TakeRequest{
			Key:   key,
			Limit: limit,
			Cost:  1,
		})
		require.NoError(t, err)
		assert.True(t, decision.Allowed)
	}
	assert.Len(t, store.records, 2)
}

func TestStoreReset(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	store := newStoreAt(&now)
	request := ratelimit.TakeRequest{
		Key:   "key",
		Limit: ratelimit.Limit{Rate: 1, Period: time.Minute, Burst: 1},
		Cost:  1,
	}

	decision, err := store.Take(t.Context(), request)
	require.NoError(t, err)
	assert.True(t, decision.Allowed)
	decision, err = store.Take(t.Context(), request)
	require.NoError(t, err)
	assert.False(t, decision.Allowed)

	require.NoError(t, store.Reset(t.Context(), request.Key))
	decision, err = store.Take(t.Context(), request)
	require.NoError(t, err)
	assert.True(t, decision.Allowed)
}

func TestStoreCleansRecoveredRecordsAcrossKeys(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	store := newStoreAt(&now)
	ctx := t.Context()

	_, err := store.Take(ctx, ratelimit.TakeRequest{
		Key:   "recovered",
		Limit: ratelimit.Limit{Rate: 1, Period: time.Second, Burst: 1},
		Cost:  1,
	})
	require.NoError(t, err)
	_, err = store.Take(ctx, ratelimit.TakeRequest{
		Key:   "active",
		Limit: ratelimit.Limit{Rate: 1, Period: time.Hour, Burst: 1},
		Cost:  1,
	})
	require.NoError(t, err)

	now = now.Add(time.Second)
	_, err = store.Take(ctx, ratelimit.TakeRequest{
		Key:   "new",
		Limit: ratelimit.Limit{Rate: 1, Period: time.Minute, Burst: 1},
		Cost:  1,
	})
	require.NoError(t, err)

	assert.NotContains(t, store.records, "recovered")
	assert.Contains(t, store.records, "active")
	assert.Contains(t, store.records, "new")
}

func TestStoreRoundsEmissionIntervalUpToMicrosecond(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	store := newStoreAt(&now)
	request := ratelimit.TakeRequest{
		Key:   "key",
		Limit: ratelimit.Limit{Rate: 3, Period: 10 * time.Microsecond, Burst: 1},
		Cost:  1,
	}

	decision, err := store.Take(t.Context(), request)
	require.NoError(t, err)
	assert.True(t, decision.Allowed)
	assert.Equal(t, 4*time.Microsecond, decision.ResetAfter)

	now = now.Add(3 * time.Microsecond)
	decision, err = store.Take(t.Context(), request)
	require.NoError(t, err)
	assert.False(t, decision.Allowed)
	assert.Equal(t, time.Microsecond, decision.RetryAfter)

	now = time.Unix(1_700_000_000, 0)
	store = newStoreAt(&now)
	decision, err = store.Take(t.Context(), ratelimit.TakeRequest{
		Key: "fractional-microsecond",
		Limit: ratelimit.Limit{
			Rate:   1,
			Period: 1500 * time.Nanosecond,
			Burst:  1,
		},
		Cost: 1,
	})
	require.NoError(t, err)
	assert.Equal(t, 2*time.Microsecond, decision.ResetAfter)
}

func TestStoreConcurrentTakeDoesNotExceedBurst(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	store := newStoreAt(&now)
	const (
		callers = 32
		burst   = 5
	)
	request := ratelimit.TakeRequest{
		Key:   "key",
		Limit: ratelimit.Limit{Rate: 1, Period: time.Minute, Burst: burst},
		Cost:  1,
	}

	var allowed atomic.Int32
	var wg sync.WaitGroup
	errs := make(chan error, callers)
	wg.Add(callers)
	for range callers {
		go func() {
			defer wg.Done()
			decision, err := store.Take(t.Context(), request)
			if err != nil {
				errs <- err
				return
			}
			if decision.Allowed {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
	assert.Equal(t, int32(burst), allowed.Load())
}

func TestStoreContext(t *testing.T) {
	store := New()
	request := ratelimit.TakeRequest{
		Key:   "key",
		Limit: ratelimit.PerSecond(1),
		Cost:  1,
	}

	//nolint:staticcheck // Verifies the documented nil Context behavior.
	_, err := store.Take(nil, request)
	assert.ErrorIs(t, err, ratelimit.ErrInvalidContext)
	//nolint:staticcheck // Verifies the documented nil Context behavior.
	assert.ErrorIs(t, store.Reset(nil, request.Key), ratelimit.ErrInvalidContext)

	expected := errors.New("canceled")
	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(expected)
	_, err = store.Take(ctx, request)
	assert.ErrorIs(t, err, expected)
	assert.ErrorIs(t, store.Reset(ctx, request.Key), expected)
}

func newStoreAt(now *time.Time) *Store {
	store := New()
	store.now = func() time.Time {
		return *now
	}
	return store
}
