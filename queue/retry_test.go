package queue

import (
	"errors"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetryPolicy_NoRetry(t *testing.T) {
	t.Parallel()

	delay, retry := NoRetry().NextDelay(&Task{Attempt: 1}, errors.New("failed"))

	assert.False(t, retry)
	assert.Zero(t, delay)
}

func TestRetryPolicy_FixedRetry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		policy    RetryPolicy
		task      *Task
		wantDelay time.Duration
		wantRetry bool
	}{
		{
			name:      "retries below max attempts",
			policy:    FixedRetry(3, 2*time.Second),
			task:      &Task{Attempt: 1},
			wantDelay: 2 * time.Second,
			wantRetry: true,
		},
		{
			name:      "stops at max attempts",
			policy:    FixedRetry(3, 2*time.Second),
			task:      &Task{Attempt: 3},
			wantRetry: false,
		},
		{
			name:      "does not retry nil task",
			policy:    FixedRetry(3, 2*time.Second),
			task:      nil,
			wantRetry: false,
		},
		{
			name:      "normalizes invalid max attempts",
			policy:    FixedRetry(0, 2*time.Second),
			task:      &Task{Attempt: 1},
			wantRetry: false,
		},
		{
			name:      "normalizes negative delay",
			policy:    FixedRetry(3, -time.Second),
			task:      &Task{Attempt: 1},
			wantDelay: 0,
			wantRetry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			delay, retry := tt.policy.NextDelay(tt.task, errors.New("failed"))

			assert.Equal(t, tt.wantRetry, retry)
			assert.Equal(t, tt.wantDelay, delay)
		})
	}
}

func TestRetryPolicy_ExponentialRetry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		policy    RetryPolicy
		task      *Task
		wantDelay time.Duration
		wantRetry bool
	}{
		{
			name:      "uses base delay on first attempt",
			policy:    ExponentialRetry(4, time.Second, 0),
			task:      &Task{Attempt: 1},
			wantDelay: time.Second,
			wantRetry: true,
		},
		{
			name:      "doubles delay after each attempt",
			policy:    ExponentialRetry(4, time.Second, 0),
			task:      &Task{Attempt: 3},
			wantDelay: 4 * time.Second,
			wantRetry: true,
		},
		{
			name:      "caps delay at max delay",
			policy:    ExponentialRetry(5, time.Second, 2500*time.Millisecond),
			task:      &Task{Attempt: 3},
			wantDelay: 2500 * time.Millisecond,
			wantRetry: true,
		},
		{
			name:      "raises max delay below base delay",
			policy:    ExponentialRetry(5, time.Second, 500*time.Millisecond),
			task:      &Task{Attempt: 2},
			wantDelay: time.Second,
			wantRetry: true,
		},
		{
			name:      "stops at max attempts",
			policy:    ExponentialRetry(4, time.Second, 0),
			task:      &Task{Attempt: 4},
			wantRetry: false,
		},
		{
			name:      "does not retry nil task",
			policy:    ExponentialRetry(4, time.Second, 0),
			task:      nil,
			wantRetry: false,
		},
		{
			name:      "normalizes invalid max attempts",
			policy:    ExponentialRetry(0, time.Second, 0),
			task:      &Task{Attempt: 1},
			wantRetry: false,
		},
		{
			name:      "normalizes negative base delay",
			policy:    ExponentialRetry(4, -time.Second, 0),
			task:      &Task{Attempt: 2},
			wantDelay: 0,
			wantRetry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			delay, retry := tt.policy.NextDelay(tt.task, errors.New("failed"))

			assert.Equal(t, tt.wantRetry, retry)
			assert.Equal(t, tt.wantDelay, delay)
		})
	}
}

func TestRetryPolicy_JitterRetry(t *testing.T) {
	t.Parallel()

	delay, retry := JitterRetry(FixedRetry(3, time.Second), 100*time.Millisecond).
		NextDelay(&Task{Attempt: 1}, errors.New("failed"))

	assert.True(t, retry)
	assert.GreaterOrEqual(t, delay, time.Second)
	assert.LessOrEqual(t, delay, 1100*time.Millisecond)
}

func TestRetryPolicy_JitterRetryNoRetry(t *testing.T) {
	t.Parallel()

	delay, retry := JitterRetry(NoRetry(), time.Second).
		NextDelay(&Task{Attempt: 1}, errors.New("failed"))

	assert.False(t, retry)
	assert.Zero(t, delay)
}

func TestRetryPolicy_JitterRetryNormalizesInputs(t *testing.T) {
	t.Parallel()

	delay, retry := JitterRetry(FixedRetry(3, time.Second), -time.Second).
		NextDelay(&Task{Attempt: 1}, errors.New("failed"))

	assert.True(t, retry)
	assert.Equal(t, time.Second, delay)

	delay, retry = JitterRetry(nil, time.Second).
		NextDelay(&Task{Attempt: 1}, errors.New("failed"))

	assert.False(t, retry)
	assert.Zero(t, delay)
}

func TestRetryPolicy_JitterRetryAllowsMaxDuration(t *testing.T) {
	t.Parallel()

	delay, retry := JitterRetry(FixedRetry(3, 0), time.Duration(math.MaxInt64)).
		NextDelay(&Task{Attempt: 1}, errors.New("failed"))

	assert.True(t, retry)
	assert.GreaterOrEqual(t, delay, time.Duration(0))
}

func TestRetryPolicy_JitterRetryConcurrent(t *testing.T) {
	t.Parallel()

	policy := JitterRetry(FixedRetry(3, time.Second), time.Millisecond)
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			delay, retry := policy.NextDelay(&Task{Attempt: 1}, errors.New("failed"))
			assert.True(t, retry)
			assert.GreaterOrEqual(t, delay, time.Second)
			assert.LessOrEqual(t, delay, time.Second+time.Millisecond)
		})
	}
	wg.Wait()
}
