package poll_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
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
	attempts := 0
	err := poll.Until(t.Context(), time.Millisecond, func(context.Context) (bool, error) {
		attempts++
		return attempts == 3, nil
	})

	require.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

func TestUntilReturnsConditionError(t *testing.T) {
	expectedErr := errors.New("condition failed")
	err := poll.Until(t.Context(), time.Millisecond, func(context.Context) (bool, error) {
		return false, expectedErr
	})

	require.ErrorIs(t, err, expectedErr)
}

func TestUntilReturnsCancellationDuringWait(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	conditionCalled := make(chan struct{})

	go func() {
		<-conditionCalled
		cancel()
	}()

	started := time.Now()
	err := poll.Until(ctx, time.Hour, func(context.Context) (bool, error) {
		close(conditionCalled)
		return false, nil
	})

	require.ErrorIs(t, err, context.Canceled)
	assert.Less(t, time.Since(started), time.Second)
}

func TestUntilReturnsDeadlineExceeded(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()

	err := poll.Until(ctx, time.Hour, func(context.Context) (bool, error) {
		return false, nil
	})

	require.ErrorIs(t, err, context.DeadlineExceeded)
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
	const interval = 30 * time.Millisecond

	attempts := 0
	var firstReturnedAt time.Time
	var secondCalledAt time.Time
	err := poll.Until(t.Context(), interval, func(context.Context) (bool, error) {
		attempts++
		if attempts == 1 {
			time.Sleep(20 * time.Millisecond)
			firstReturnedAt = time.Now()
			return false, nil
		}
		secondCalledAt = time.Now()
		return true, nil
	})

	require.NoError(t, err)
	assert.GreaterOrEqual(t, secondCalledAt.Sub(firstReturnedAt), interval)
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
	attempts := 0
	value, err := poll.UntilValue(
		t.Context(),
		time.Millisecond,
		func(context.Context) (string, bool, error) {
			attempts++
			value := "pending"
			if attempts == 2 {
				value = "ready"
			}
			return value, attempts == 2, nil
		},
	)

	require.NoError(t, err)
	assert.Equal(t, "ready", value)
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
	ctx, cancel := context.WithCancel(t.Context())
	conditionCalled := make(chan struct{})

	go func() {
		<-conditionCalled
		cancel()
	}()

	value, err := poll.UntilValue(ctx, time.Hour, func(context.Context) (int, bool, error) {
		close(conditionCalled)
		return 42, false, nil
	})

	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 42, value)
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
