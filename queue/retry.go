package queue

import (
	"errors"
	"math"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"
)

// ErrRetryExhausted is used as the dead-letter reason when a task has no retries left.
var ErrRetryExhausted = errors.New("queue: retry exhausted")

// RetryPolicy decides whether a failed task should be retried and after what delay.
type RetryPolicy interface {
	// NextDelay returns the next retry delay and whether another retry should happen.
	NextDelay(task *Task, err error) (time.Duration, bool)
}

type jitterRetry struct {
	policy    RetryPolicy
	maxJitter time.Duration
	mu        sync.Mutex
	rand      *rand.Rand
}

var jitterSeed atomic.Uint64

// JitterRetry wraps policy and adds bounded random jitter to retry delays.
func JitterRetry(policy RetryPolicy, maxJitter time.Duration) RetryPolicy {
	if policy == nil {
		policy = NoRetry()
	}
	if maxJitter < 0 {
		maxJitter = 0
	}
	return &jitterRetry{
		policy:    policy,
		maxJitter: maxJitter,
		rand:      newJitterRand(),
	}
}

func newJitterRand() *rand.Rand {
	seed := uint64(time.Now().UnixNano())
	sequence := jitterSeed.Add(1)
	return rand.New(rand.NewPCG(seed, seed^sequence))
}

func (r *jitterRetry) NextDelay(task *Task, err error) (time.Duration, bool) {
	delay, retry := r.policy.NextDelay(task, err)
	if !retry || r.maxJitter <= 0 {
		return delay, retry
	}
	r.mu.Lock()
	jitter := time.Duration(r.rand.Uint64N(uint64(r.maxJitter) + 1))
	r.mu.Unlock()
	if jitter > 0 && delay > time.Duration(math.MaxInt64)-jitter {
		return time.Duration(math.MaxInt64), true
	}
	return delay + jitter, true
}

type noRetry struct{}

// NoRetry returns a retry policy that dead-letters every failed task immediately.
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

// FixedRetry returns a retry policy with a constant delay between attempts.
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

func (r fixedRetry) NextDelay(task *Task, _ error) (time.Duration, bool) {
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

// ExponentialRetry returns a retry policy whose delay doubles after each failed attempt.
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

func (r exponentialRetry) NextDelay(task *Task, _ error) (time.Duration, bool) {
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
