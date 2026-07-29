package ratelimit

import (
	"context"
	"time"
)

// Decision reports the result of consuming capacity from a [Limiter].
type Decision struct {
	// Limit is the policy used to produce this decision.
	Limit Limit
	// Allowed reports whether the requested cost was consumed.
	Allowed bool
	// Remaining is the maximum number of units that can be consumed
	// immediately after this decision.
	Remaining int
	// RetryAfter is the minimum delay before the rejected cost can be accepted.
	// It is zero when Allowed is true.
	RetryAfter time.Duration
	// ResetAfter is the time until the key has recovered its full burst.
	ResetAfter time.Duration
}

// TakeRequest contains the data required for one atomic capacity decision.
type TakeRequest struct {
	// Key identifies an independent rate-limit state.
	Key string
	// Limit is the policy applied to the key.
	Limit Limit
	// Cost is the number of units to consume atomically.
	Cost int
}

// Store atomically stores and updates rate-limit state.
//
// Implementations must be safe for concurrent use and linearize [Store.Take]
// calls for the same key. A rejected Take must not consume capacity. Methods
// return [ErrInvalidContext] when ctx is nil and honor context cancellation.
type Store interface {
	// Take atomically decides whether request.Cost units can be consumed.
	//
	// Take returns a decision with a nil error when capacity is unavailable.
	// Requests are normally created and validated by [Limiter]; direct callers
	// must provide a non-empty key, a valid limit, and a cost between one and
	// the limit's burst.
	Take(context.Context, TakeRequest) (Decision, error)

	// Reset removes the stored state for key.
	Reset(context.Context, string) error
}
