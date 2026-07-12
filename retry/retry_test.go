package retry

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDo(t *testing.T) {
	t.Parallel()

	t.Run("immediate success", func(t *testing.T) {
		t.Parallel()
		var attempts int
		err := Do(t.Context(), func(context.Context) error {
			attempts++
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, 1, attempts)
	})

	t.Run("eventual success", func(t *testing.T) {
		t.Parallel()
		var attempts int
		err := Do(t.Context(), func(context.Context) error {
			attempts++
			if attempts < 3 {
				return assert.AnError
			}
			return nil
		}, WithBackoff(NoBackoff()))
		require.NoError(t, err)
		assert.Equal(t, 3, attempts)
	})

	t.Run("exhaustion returns last error", func(t *testing.T) {
		t.Parallel()
		firstErr := errors.New("first")
		lastErr := errors.New("last")
		var attempts int
		err := Do(t.Context(), func(context.Context) error {
			attempts++
			if attempts == 1 {
				return firstErr
			}
			return lastErr
		}, WithMaxAttempts(2), WithBackoff(NoBackoff()))
		require.ErrorIs(t, err, lastErr)
		assert.Equal(t, 2, attempts)
	})
}

func TestDoValue(t *testing.T) {
	t.Parallel()

	t.Run("returns successful value", func(t *testing.T) {
		t.Parallel()
		var attempts int
		value, err := DoValue(t.Context(), func(context.Context) (string, error) {
			attempts++
			if attempts == 1 {
				return "partial", assert.AnError
			}
			return "complete", nil
		}, WithBackoff(NoBackoff()))
		require.NoError(t, err)
		assert.Equal(t, "complete", value)
	})

	t.Run("preserves final failed value", func(t *testing.T) {
		t.Parallel()
		value, err := DoValue(t.Context(), func(context.Context) (string, error) {
			return "partial", assert.AnError
		}, WithMaxAttempts(1))
		require.ErrorIs(t, err, assert.AnError)
		assert.Equal(t, "partial", value)
	})
}

func TestDoContext(t *testing.T) {
	t.Parallel()

	t.Run("already canceled", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		var called bool
		err := Do(ctx, func(context.Context) error {
			called = true
			return nil
		})
		require.ErrorIs(t, err, context.Canceled)
		assert.False(t, called)
	})

	t.Run("canceled while waiting", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		var attempts int
		err := Do(
			ctx,
			func(context.Context) error {
				attempts++
				return assert.AnError
			},
			WithBackoff(Fixed(time.Hour)),
			WithNotify(func(context.Context, Event) { cancel() }),
		)
		require.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, 1, attempts)
	})

	t.Run("deadline while waiting", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
		defer cancel()
		err := Do(ctx, func(context.Context) error {
			return assert.AnError
		}, WithBackoff(Fixed(time.Hour)))
		require.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("operation cancellation wins", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		var attempts int
		err := Do(ctx, func(context.Context) error {
			attempts++
			cancel()
			return assert.AnError
		}, WithBackoff(NoBackoff()))
		require.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, 1, attempts)
	})
}

func TestDoRetryPredicate(t *testing.T) {
	t.Parallel()

	t.Run("rejects error", func(t *testing.T) {
		t.Parallel()
		var attempts int
		err := Do(t.Context(), func(context.Context) error {
			attempts++
			return assert.AnError
		}, WithRetryIf(func(error) bool { return false }))
		require.ErrorIs(t, err, assert.AnError)
		assert.Equal(t, 1, attempts)
	})

	t.Run("is not called after final attempt", func(t *testing.T) {
		t.Parallel()
		var predicateCalls int
		err := Do(
			t.Context(), func(context.Context) error {
				return assert.AnError
			},
			WithMaxAttempts(2),
			WithBackoff(NoBackoff()),
			WithRetryIf(func(error) bool {
				predicateCalls++
				return true
			}),
		)
		require.ErrorIs(t, err, assert.AnError)
		assert.Equal(t, 1, predicateCalls)
	})

	t.Run("does not retry context errors by default", func(t *testing.T) {
		t.Parallel()
		var attempts int
		err := Do(t.Context(), func(context.Context) error {
			attempts++
			return fmt.Errorf("request: %w", context.DeadlineExceeded)
		})
		require.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Equal(t, 1, attempts)
	})
}

func TestPermanent(t *testing.T) {
	t.Parallel()

	assert.NoError(t, Permanent(nil))
	marked := Permanent(assert.AnError)
	require.ErrorIs(t, marked, assert.AnError)

	var attempts int
	err := Do(t.Context(), func(context.Context) error {
		attempts++
		return Permanent(assert.AnError)
	})
	require.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, assert.AnError, err)
	assert.Equal(t, 1, attempts)
}

func TestAfter(t *testing.T) {
	t.Parallel()

	assert.NoError(t, After(time.Second, nil))
	marked := After(time.Second, assert.AnError)
	require.ErrorIs(t, marked, assert.AnError)

	var (
		attempts int
		event    Event
	)
	err := Do(
		t.Context(), func(context.Context) error {
			attempts++
			if attempts == 1 {
				return After(0, assert.AnError)
			}
			return nil
		},
		WithBackoff(Fixed(time.Hour)),
		WithNotify(func(_ context.Context, got Event) { event = got }),
	)
	require.NoError(t, err)
	assert.Equal(t, 2, attempts)
	assert.Zero(t, event.Delay)
	assert.Equal(t, assert.AnError, event.Err)
}

func TestNotify(t *testing.T) {
	t.Parallel()

	var events []Event
	err := Do(
		t.Context(), func(context.Context) error {
			return assert.AnError
		},
		WithMaxAttempts(3),
		WithBackoff(Linear(time.Nanosecond)),
		WithNotify(func(_ context.Context, event Event) {
			events = append(events, event)
		}),
	)
	require.ErrorIs(t, err, assert.AnError)
	require.Len(t, events, 2)
	assert.Equal(t, Event{Attempt: 1, MaxAttempts: 3, Err: assert.AnError, Delay: time.Nanosecond}, events[0])
	assert.Equal(t, Event{Attempt: 2, MaxAttempts: 3, Err: assert.AnError, Delay: 2 * time.Nanosecond}, events[1])
}

func TestBackoffIsNotCalledAfterFinalAttempt(t *testing.T) {
	t.Parallel()

	var calls int
	err := Do(
		t.Context(), func(context.Context) error {
			return assert.AnError
		},
		WithMaxAttempts(1),
		WithBackoff(func(int) time.Duration {
			calls++
			return 0
		}),
	)
	require.ErrorIs(t, err, assert.AnError)
	assert.Zero(t, calls)
}

func TestDoPanicsOnInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "nil context",
			fn: func() {
				_ = Do(nil, func(context.Context) error { return nil }) //nolint:staticcheck // Verify the documented nil-context panic.
			},
		},
		{name: "nil operation", fn: func() { _ = Do(t.Context(), nil) }},
		{name: "nil value operation", fn: func() { _, _ = DoValue[string](t.Context(), nil) }},
		{name: "zero attempts", fn: func() { WithMaxAttempts(0) }},
		{name: "nil backoff", fn: func() { WithBackoff(nil) }},
		{name: "nil predicate", fn: func() { WithRetryIf(nil) }},
		{name: "negative retry after", fn: func() { _ = After(-1, assert.AnError) }},
		{
			name: "negative custom backoff",
			fn: func() {
				_ = Do(
					t.Context(), func(context.Context) error { return assert.AnError },
					WithBackoff(func(int) time.Duration { return -1 }),
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Panics(t, tt.fn)
		})
	}
}

func TestSharedBackoffAcrossConcurrentRetries(t *testing.T) {
	t.Parallel()

	backoff := Jitter(Fixed(time.Nanosecond), time.Nanosecond)
	var completed atomic.Int64
	t.Run("concurrent", func(t *testing.T) {
		t.Parallel()
		for range 20 {
			t.Run("retry", func(t *testing.T) {
				t.Parallel()
				var attempts int
				err := Do(t.Context(), func(context.Context) error {
					attempts++
					if attempts == 1 {
						return assert.AnError
					}
					completed.Add(1)
					return nil
				}, WithBackoff(backoff))
				require.NoError(t, err)
			})
		}
	})
	t.Cleanup(func() {
		assert.Equal(t, int64(20), completed.Load())
	})
}
