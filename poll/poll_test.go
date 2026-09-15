package poll_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-fries/fries/poll/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUntilCanceledContextDoesNotCallCondition(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	called := false
	err := poll.Until(ctx, time.Millisecond, func(context.Context) (bool, error) {
		called = true
		return true, nil
	})

	require.ErrorIs(t, err, context.Canceled)
	assert.False(t, called)
}

func TestUntilCallsConditionImmediately(t *testing.T) {
	called := false
	err := poll.Until(t.Context(), time.Hour, func(context.Context) (bool, error) {
		called = true
		return true, nil
	})

	require.NoError(t, err)
	assert.True(t, called)
}

func TestUntilPollsUntilComplete(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const interval = time.Second
		started := time.Now()
		var calledAt []time.Duration
		err := poll.Until(t.Context(), interval, func(context.Context) (bool, error) {
			calledAt = append(calledAt, time.Since(started))
			return len(calledAt) == 3, nil
		})

		require.NoError(t, err)
		assert.Equal(t, []time.Duration{0, interval, 2 * interval}, calledAt)
		assert.Equal(t, 2*interval, time.Since(started))
	})
}

func TestUntilReturnsConditionError(t *testing.T) {
	expectedErr := errors.New("condition failed")
	err := poll.Until(t.Context(), time.Millisecond, func(context.Context) (bool, error) {
		return false, expectedErr
	})

	require.ErrorIs(t, err, expectedErr)
}

func TestUntilReturnsCancellationDuringWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		started := time.Now()
		result := make(chan error, 1)
		var attempts atomic.Int32
		go func() {
			result <- poll.Until(ctx, time.Hour, func(context.Context) (bool, error) {
				attempts.Add(1)
				return false, nil
			})
		}()

		synctest.Wait()
		assert.Equal(t, int32(1), attempts.Load())
		require.Empty(t, result, "polling returned before cancellation")

		cancel()
		synctest.Wait()
		require.Len(t, result, 1, "cancellation did not interrupt the polling interval")
		require.ErrorIs(t, <-result, context.Canceled)
		assert.Equal(t, int32(1), attempts.Load())
		assert.Zero(t, time.Since(started))
	})
}

func TestUntilReturnsDeadlineExceeded(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const timeout = time.Second
		ctx, cancel := context.WithTimeout(t.Context(), timeout)
		defer cancel()
		started := time.Now()
		result := make(chan error, 1)
		var attempts atomic.Int32
		go func() {
			result <- poll.Until(ctx, time.Hour, func(context.Context) (bool, error) {
				attempts.Add(1)
				return false, nil
			})
		}()

		synctest.Wait()
		assert.Equal(t, int32(1), attempts.Load())
		require.Empty(t, result)

		time.Sleep(timeout - time.Nanosecond)
		synctest.Wait()
		assert.NoError(t, ctx.Err())
		assert.Equal(t, int32(1), attempts.Load())
		require.Empty(t, result, "polling returned before its deadline")

		time.Sleep(time.Nanosecond)
		synctest.Wait()
		require.Len(t, result, 1, "deadline did not interrupt the polling interval")
		require.ErrorIs(t, <-result, context.DeadlineExceeded)
		assert.Equal(t, int32(1), attempts.Load())
		assert.Equal(t, timeout, time.Since(started))
	})
}

func TestUntilContextErrorTakesPrecedence(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	expectedErr := errors.New("condition failed")

	err := poll.Until(ctx, time.Millisecond, func(context.Context) (bool, error) {
		cancel()
		return false, expectedErr
	})

	require.ErrorIs(t, err, context.Canceled)
	assert.NotErrorIs(t, err, expectedErr)
}

func TestUntilWaitsAfterConditionReturns(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const (
			interval      = 3 * time.Second
			conditionTime = 2 * time.Second
		)
		started := time.Now()
		var attempts int
		var firstReturnedAt time.Time
		var secondCalledAt time.Time
		err := poll.Until(t.Context(), interval, func(context.Context) (bool, error) {
			attempts++
			if attempts == 1 {
				time.Sleep(conditionTime)
				firstReturnedAt = time.Now()
				return false, nil
			}
			secondCalledAt = time.Now()
			return true, nil
		})

		require.NoError(t, err)
		assert.Equal(t, 2, attempts)
		assert.Equal(t, conditionTime, firstReturnedAt.Sub(started))
		assert.Equal(t, interval, secondCalledAt.Sub(firstReturnedAt))
		assert.Equal(t, conditionTime+interval, time.Since(started))
	})
}

func TestUntilPanicsForNilCondition(t *testing.T) {
	assert.PanicsWithValue(t, "poll: nil condition", func() {
		_ = poll.Until(t.Context(), time.Millisecond, nil)
	})
}

func TestUntilPanicsForNonPositiveInterval(t *testing.T) {
	condition := func(context.Context) (bool, error) {
		return true, nil
	}

	for _, interval := range []time.Duration{0, -time.Millisecond} {
		assert.PanicsWithValue(t, "poll: interval must be greater than zero", func() {
			_ = poll.Until(t.Context(), interval, condition)
		})
	}
}

func TestUntilValueReturnsCompletedValue(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const interval = time.Second
		started := time.Now()
		var calledAt []time.Duration
		value, err := poll.UntilValue(
			t.Context(),
			interval,
			func(context.Context) (string, bool, error) {
				calledAt = append(calledAt, time.Since(started))
				if len(calledAt) == 2 {
					return "ready", true, nil
				}
				return "pending", false, nil
			},
		)

		require.NoError(t, err)
		assert.Equal(t, "ready", value)
		assert.Equal(t, []time.Duration{0, interval}, calledAt)
		assert.Equal(t, interval, time.Since(started))
	})
}

func TestUntilValueReturnsZeroValueWhenCanceledBeforeCondition(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	value, err := poll.UntilValue(ctx, time.Millisecond, func(context.Context) (int, bool, error) {
		return 42, true, nil
	})

	require.ErrorIs(t, err, context.Canceled)
	assert.Zero(t, value)
}

func TestUntilValueReturnsValueWithConditionError(t *testing.T) {
	expectedErr := errors.New("condition failed")
	value, err := poll.UntilValue(
		t.Context(),
		time.Millisecond,
		func(context.Context) (string, bool, error) {
			return "latest", false, expectedErr
		},
	)

	require.ErrorIs(t, err, expectedErr)
	assert.Equal(t, "latest", value)
}

func TestUntilValueReturnsLatestValueWhenCanceledDuringWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const interval = time.Second
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		type result struct {
			value int
			err   error
		}
		results := make(chan result, 1)
		var attempts atomic.Int32
		go func() {
			value, err := poll.UntilValue(ctx, interval, func(context.Context) (int, bool, error) {
				return 40 + int(attempts.Add(1)), false, nil
			})
			results <- result{value: value, err: err}
		}()

		synctest.Wait()
		assert.Equal(t, int32(1), attempts.Load())
		require.Empty(t, results)

		time.Sleep(interval)
		synctest.Wait()
		assert.Equal(t, int32(2), attempts.Load())
		require.Empty(t, results, "polling returned before cancellation")

		canceledAt := time.Now()
		cancel()
		synctest.Wait()
		require.Len(t, results, 1, "cancellation did not interrupt the polling interval")
		got := <-results
		require.ErrorIs(t, got.err, context.Canceled)
		assert.Equal(t, 42, got.value)
		assert.Equal(t, int32(2), attempts.Load())
		assert.Zero(t, time.Since(canceledAt))
	})
}

func TestUntilValueReturnsLatestValueWhenCanceledDuringCondition(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	expectedErr := errors.New("condition failed")

	value, err := poll.UntilValue(ctx, time.Millisecond, func(context.Context) (int, bool, error) {
		cancel()
		return 42, false, expectedErr
	})

	require.ErrorIs(t, err, context.Canceled)
	assert.NotErrorIs(t, err, expectedErr)
	assert.Equal(t, 42, value)
}

func TestUntilValuePanicsForNilCondition(t *testing.T) {
	assert.PanicsWithValue(t, "poll: nil condition", func() {
		_, _ = poll.UntilValue[int](t.Context(), time.Millisecond, nil)
	})
}

func TestUntilConcurrentSafe(t *testing.T) {
	const callers = 16

	var completed atomic.Int64
	var waitGroup sync.WaitGroup
	waitGroup.Add(callers)
	for range callers {
		go func() {
			defer waitGroup.Done()

			attempts := 0
			err := poll.Until(t.Context(), time.Microsecond, func(context.Context) (bool, error) {
				attempts++
				return attempts == 2, nil
			})
			if err == nil && attempts == 2 {
				completed.Add(1)
			}
		}()
	}
	waitGroup.Wait()

	assert.Equal(t, int64(callers), completed.Load())
}
