package idempotency

import "errors"

var (
	// ErrInvalidContext indicates that a required context is nil.
	ErrInvalidContext = errors.New("idempotency: invalid context")
	// ErrInvalidKey indicates that an idempotency key is empty.
	ErrInvalidKey = errors.New("idempotency: invalid key")
	// ErrInProgress indicates that another caller owns an active execution
	// claim for the key.
	ErrInProgress = errors.New("idempotency: operation in progress")
	// ErrKeyConflict indicates that a key was reused with a different
	// fingerprint.
	ErrKeyConflict = errors.New("idempotency: key reused with different input")
	// ErrClaimLost indicates that an execution claim expired or was replaced
	// before it could be completed or aborted.
	ErrClaimLost = errors.New("idempotency: execution claim lost")
)
