package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-fries/fries/retry/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorker_UsesConfiguredBackoff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		backoff  retry.Backoff
		attempt  int
		assertFn func(*testing.T, time.Duration)
	}{
		{
			name:    "fixed",
			backoff: retry.Fixed(2 * time.Second),
			attempt: 1,
			assertFn: func(t *testing.T, delay time.Duration) {
				assert.Equal(t, 2*time.Second, delay)
			},
		},
		{
			name:    "linear",
			backoff: retry.Linear(time.Second),
			attempt: 2,
			assertFn: func(t *testing.T, delay time.Duration) {
				assert.Equal(t, 2*time.Second, delay)
			},
		},
		{
			name:    "exponential",
			backoff: retry.Exponential(time.Second, 0),
			attempt: 3,
			assertFn: func(t *testing.T, delay time.Duration) {
				assert.Equal(t, 4*time.Second, delay)
			},
		},
		{
			name: "jitter",
			backoff: retry.Jitter(
				retry.Fixed(time.Second),
				100*time.Millisecond,
			),
			attempt: 1,
			assertFn: func(t *testing.T, delay time.Duration) {
				assert.GreaterOrEqual(t, delay, time.Second)
				assert.LessOrEqual(t, delay, 1100*time.Millisecond)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			retried := make(chan time.Duration, 1)
			worker := NewWorker(
				newTestQueue(),
				Handle("fail", HandlerFunc(func(context.Context, *Task) error {
					return errors.New("failed")
				})),
				WithMaxAttempts(5),
				WithBackoff(tt.backoff),
			)
			delivery := &recordingDelivery{
				task: &Task{Type: "fail", Attempt: tt.attempt},
				retry: func(_ context.Context, delay time.Duration) error {
					retried <- delay
					return nil
				},
			}

			err := worker.process(t.Context(), delivery)

			require.NoError(t, err)
			tt.assertFn(t, <-retried)
		})
	}
}

func TestWorker_NormalizesNegativeBackoffDelay(t *testing.T) {
	t.Parallel()

	retried := make(chan time.Duration, 1)
	worker := NewWorker(
		newTestQueue(),
		Handle("fail", HandlerFunc(func(context.Context, *Task) error {
			return errors.New("failed")
		})),
		WithBackoff(func(int) time.Duration { return -time.Second }),
	)
	delivery := &recordingDelivery{
		task: &Task{Type: "fail", Attempt: 1},
		retry: func(_ context.Context, delay time.Duration) error {
			retried <- delay
			return nil
		},
	}

	err := worker.process(t.Context(), delivery)

	require.NoError(t, err)
	assert.Zero(t, <-retried)
}

func TestWorker_RetryIfDeadLettersWithHandlerError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("permanent failure")
	deadLettered := make(chan string, 1)
	worker := NewWorker(
		newTestQueue(),
		Handle("fail", HandlerFunc(func(context.Context, *Task) error {
			return wantErr
		})),
		WithRetryIf(func(task *Task, err error) bool {
			assert.Equal(t, "fail", task.Type)
			assert.ErrorIs(t, err, wantErr)
			return false
		}),
	)
	delivery := &recordingDelivery{
		task: &Task{Type: "fail", Attempt: 1},
		deadLetter: func(_ context.Context, reason string) error {
			deadLettered <- reason
			return nil
		},
	}

	err := worker.process(t.Context(), delivery)

	require.NoError(t, err)
	assert.Equal(t, wantErr.Error(), <-deadLettered)
}

func TestWorker_DoesNotCallRetryIfAfterFinalAttempt(t *testing.T) {
	t.Parallel()

	deadLettered := make(chan string, 1)
	worker := NewWorker(
		newTestQueue(),
		Handle("fail", HandlerFunc(func(context.Context, *Task) error {
			return errors.New("failed")
		})),
		WithMaxAttempts(2),
		WithRetryIf(func(*Task, error) bool {
			require.Fail(t, "retry predicate called after final attempt")
			return true
		}),
	)
	delivery := &recordingDelivery{
		task: &Task{Type: "fail", Attempt: 2},
		deadLetter: func(_ context.Context, reason string) error {
			deadLettered <- reason
			return nil
		},
	}

	err := worker.process(t.Context(), delivery)

	require.NoError(t, err)
	assert.Contains(t, <-deadLettered, ErrRetryExhausted.Error())
}

func TestWorker_RetryAfterBypassesRetryIfAndBackoff(t *testing.T) {
	t.Parallel()

	retried := make(chan time.Duration, 1)
	worker := NewWorker(
		newTestQueue(),
		Handle("rate_limited", HandlerFunc(func(context.Context, *Task) error {
			return RetryAfter(5 * time.Second)
		})),
		WithMaxAttempts(2),
		WithRetryIf(func(*Task, error) bool {
			require.Fail(t, "retry predicate called for RetryAfter")
			return false
		}),
		WithBackoff(func(int) time.Duration {
			require.Fail(t, "backoff called for RetryAfter")
			return 0
		}),
	)
	delivery := &recordingDelivery{
		task: &Task{Type: "rate_limited", Attempt: 1},
		retry: func(_ context.Context, delay time.Duration) error {
			retried <- delay
			return nil
		},
	}

	err := worker.process(t.Context(), delivery)

	require.NoError(t, err)
	assert.Equal(t, 5*time.Second, <-retried)
}
