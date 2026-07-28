package idempotency

import (
	"context"
	"time"
)

// BeginStatus describes the outcome of a Store.Begin operation.
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
	Key         string
	Token       string
	Fingerprint string
	TTL         time.Duration
}

// BeginResult reports the current state of an idempotency key.
//
// Result contains the stored result when Status is BeginCompleted. Executor.Do
// ignores Result, while DoValue decodes it into the requested value type.
type BeginResult struct {
	Status BeginStatus
	Result []byte
}

// CompleteRequest contains the data required to complete an execution claim.
type CompleteRequest struct {
	Key    string
	Token  string
	Result []byte
	TTL    time.Duration
}

// AbortRequest identifies an execution claim to remove after a failed Handler.
type AbortRequest struct {
	Key   string
	Token string
}

// Store atomically coordinates execution claims and completed results.
//
// Implementations must linearize operations for the same key. Complete and
// Abort must return ErrClaimLost when the supplied token no longer owns an
// active claim.
type Store interface {
	Begin(context.Context, BeginRequest) (BeginResult, error)
	Complete(context.Context, CompleteRequest) error
	Abort(context.Context, AbortRequest) error
}
