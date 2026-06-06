package queue

import (
	"errors"
	"time"
)

var ErrRetryExhausted = errors.New("queue: retry exhausted")

type RetryPolicy interface {
	NextDelay(task *Task, err error) (time.Duration, bool)
}

type noRetry struct{}

func NoRetry() RetryPolicy {
	return noRetry{}
}

func (noRetry) NextDelay(*Task, error) (time.Duration, bool) {
	return 0, false
}

type fixedRetry struct {
	maxAttempts int
	delay       time.Duration
}

func FixedRetry(maxAttempts int, delay time.Duration) RetryPolicy {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	if delay < 0 {
		delay = 0
	}
	return fixedRetry{
		maxAttempts: maxAttempts,
		delay:       delay,
	}
}

func (r fixedRetry) NextDelay(task *Task, errorErr error) (time.Duration, bool) {
	if task == nil || task.Attempt >= r.maxAttempts {
		return 0, false
	}
	return r.delay, true
}

type exponentialRetry struct {
	maxAttempts int
	baseDelay   time.Duration
	maxDelay    time.Duration
}

func ExponentialRetry(maxAttempts int, baseDelay, maxDelay time.Duration) RetryPolicy {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	if baseDelay < 0 {
		baseDelay = 0
	}
	if maxDelay > 0 && maxDelay < baseDelay {
		maxDelay = baseDelay
	}
	return exponentialRetry{
		maxAttempts: maxAttempts,
		baseDelay:   baseDelay,
		maxDelay:    maxDelay,
	}
}

func (r exponentialRetry) NextDelay(task *Task, errorErr error) (time.Duration, bool) {
	if task == nil || task.Attempt >= r.maxAttempts {
		return 0, false
	}

	delay := r.baseDelay
	for i := 1; i < task.Attempt; i++ {
		delay *= 2
		if r.maxDelay > 0 && delay >= r.maxDelay {
			return r.maxDelay, true
		}
	}
	return delay, true
}
