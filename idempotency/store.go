package idempotency

import (
	"context"
	"time"
)

// BeginStatus describes the outcome of a [Store.Begin] operation.
// The zero value is invalid.
type BeginStatus uint8

const (
	// BeginAcquired indicates that the caller created a new execution claim.
	BeginAcquired BeginStatus = iota + 1
	// BeginInProgress indicates that another caller owns an active claim.
	BeginInProgress
	// BeginCompleted indicates that the key has already completed.
	BeginCompleted
)

// BeginRequest contains the data required to claim an idempotency key.
type BeginRequest struct {
	// Key identifies the idempotent operation and must not be empty.
	Key string
	// Token uniquely identifies the caller's execution claim and must not be
	// empty.
	Token string
	// Fingerprint identifies the operation input. An empty fingerprint disables
	// conflict detection.
	Fingerprint string
	// TTL controls how long the execution claim remains active and must be
	// positive.
	TTL time.Duration
}

// BeginResult reports the current state of an idempotency key.
type BeginResult struct {
	// Status reports whether the claim was acquired, remains in progress, or
	// has already completed.
	Status BeginStatus
	// Result contains the stored result when Status is [BeginCompleted].
	// [Executor.Do] ignores Result, while [DoValue] decodes it into the requested
	// value type.
	Result []byte
}

// CompleteRequest contains the data required to complete an execution claim.
type CompleteRequest struct {
	// Key identifies the idempotent operation and must not be empty.
	Key string
	// Token identifies the execution claim to complete and must not be empty.
	Token string
	// Result contains the encoded value to persist.
	Result []byte
	// TTL controls how long the completed record remains available and must be
	// positive.
	TTL time.Duration
}

// AbortRequest identifies an execution claim to remove after a failed
// [Handler].
type AbortRequest struct {
	// Key identifies the idempotent operation and must not be empty.
	Key string
	// Token identifies the execution claim to abort and must not be empty.
	Token string
}

// Store atomically coordinates execution claims and completed results.
//
// Implementations must be safe for concurrent use and linearize operations for
// the same key. [Store.Complete] and [Store.Abort] must return [ErrClaimLost]
// when the supplied token no longer owns an active claim.
//
// Methods return [ErrInvalidContext] when ctx is nil and honor context
// cancellation.
type Store interface {
	// Begin atomically creates an execution claim for a missing or expired key,
	// or reports the current state of an existing record.
	//
	// Begin returns [ErrKeyConflict] when a non-empty fingerprint differs from
	// the fingerprint stored for the key.
	Begin(context.Context, BeginRequest) (BeginResult, error)

	// Complete atomically replaces an active claim with a completed record.
	//
	// The claim must be owned by the request token. Complete returns
	// [ErrClaimLost] when the claim expired, was replaced, or is no longer
	// active.
	Complete(context.Context, CompleteRequest) error

	// Abort atomically removes an active claim after execution fails.
	//
	// The claim must be owned by the request token. Abort returns [ErrClaimLost]
	// when the claim expired, was replaced, or is no longer active.
	Abort(context.Context, AbortRequest) error
}
