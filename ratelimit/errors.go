package ratelimit

import "errors"

var (
	// ErrInvalidContext indicates that a required context is nil.
	ErrInvalidContext = errors.New("ratelimit: invalid context")
	// ErrInvalidKey indicates that a rate-limit key is empty.
	ErrInvalidKey = errors.New("ratelimit: invalid key")
	// ErrInvalidLimit indicates that a [Limit] cannot be represented or used.
	ErrInvalidLimit = errors.New("ratelimit: invalid limit")
	// ErrInvalidCost indicates that a requested cost is not between one and the
	// configured burst.
	ErrInvalidCost = errors.New("ratelimit: invalid cost")
	// ErrNilStore indicates that a nil [Store] was supplied to [New].
	ErrNilStore = errors.New("ratelimit: nil store")
)
