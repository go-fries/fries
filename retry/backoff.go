package retry

import (
	"math"
	"math/rand/v2"
	"time"
)

// Backoff returns the delay after a failed attempt. The attempt parameter
// starts at one and identifies the execution that just failed.
//
// A Backoff must be safe for concurrent use when shared by callers.
type Backoff func(attempt int) time.Duration

// NoBackoff returns a strategy that never waits between attempts.
func NoBackoff() Backoff {
	return func(int) time.Duration {
		return 0
	}
}

// Fixed returns a strategy that always waits for delay.
//
// It panics if delay is negative.
func Fixed(delay time.Duration) Backoff {
	validateDelay("fixed delay", delay)
	return func(int) time.Duration {
		return delay
	}
}

// Linear returns a strategy whose delay is delay multiplied by the failed
// attempt number.
//
// It panics if delay is negative. Overflow is clamped to the largest duration.
func Linear(delay time.Duration) Backoff {
	validateDelay("linear delay", delay)
	return func(attempt int) time.Duration {
		if attempt < 1 || delay == 0 {
			return 0
		}
		return multiplyDuration(delay, uint64(attempt))
	}
}

// Exponential returns a strategy that starts at initial and doubles after each
// failed attempt. A maximum of zero disables the upper bound.
//
// It panics if either duration is negative, or if maximum is positive and less
// than initial. Overflow is clamped to maximum or the largest duration.
func Exponential(initial, maximum time.Duration) Backoff {
	validateDelay("exponential initial delay", initial)
	validateDelay("exponential maximum delay", maximum)
	if maximum > 0 && maximum < initial {
		panic("retry: exponential maximum delay must not be less than initial delay")
	}

	return func(attempt int) time.Duration {
		if attempt < 1 || initial == 0 {
			return 0
		}

		delay := initial
		for range attempt - 1 {
			if maximum > 0 && delay >= maximum {
				return maximum
			}
			if delay > time.Duration(math.MaxInt64/2) {
				if maximum > 0 {
					return maximum
				}
				return time.Duration(math.MaxInt64)
			}
			delay *= 2
		}
		if maximum > 0 && delay > maximum {
			return maximum
		}
		return delay
	}
}

// Jitter wraps backoff and adds a random duration in the inclusive range from
// zero through maximum to each positive delay.
//
// A nil backoff is treated as [NoBackoff]. Jitter panics if maximum is
// negative. Overflow is clamped to the largest duration.
func Jitter(backoff Backoff, maximum time.Duration) Backoff {
	if backoff == nil {
		backoff = NoBackoff()
	}
	validateDelay("jitter maximum delay", maximum)
	if maximum == 0 {
		return backoff
	}

	return func(attempt int) time.Duration {
		delay := backoff(attempt)
		if delay <= 0 {
			return delay
		}

		jitter := time.Duration(rand.Uint64N(uint64(maximum) + 1)) //nolint:gosec
		if delay > time.Duration(math.MaxInt64)-jitter {
			return time.Duration(math.MaxInt64)
		}
		return delay + jitter
	}
}

func validateDelay(name string, delay time.Duration) {
	if delay < 0 {
		panic("retry: " + name + " must not be negative")
	}
}

func multiplyDuration(delay time.Duration, multiplier uint64) time.Duration {
	if multiplier > uint64(math.MaxInt64)/uint64(delay) {
		return time.Duration(math.MaxInt64)
	}
	return delay * time.Duration(multiplier)
}
