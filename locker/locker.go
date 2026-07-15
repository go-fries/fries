package locker

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNotAcquired indicates that a lock could not be acquired because it is
	// currently held by another owner.
	ErrNotAcquired = errors.New("locker: lock not acquired")

	// ErrLeaseLost indicates that a lease has expired or is no longer owned by
	// the caller.
	ErrLeaseLost = errors.New("locker: lease lost")

	// ErrInvalidName indicates that a lock name is empty.
	ErrInvalidName = errors.New("locker: invalid lock name")

	// ErrInvalidTTL indicates that a lock TTL is not positive.
	ErrInvalidTTL = errors.New("locker: invalid lock ttl")

	// ErrInvalidToken indicates that a transferred lease token is empty.
	ErrInvalidToken = errors.New("locker: invalid lease token")

	// ErrInvalidContext indicates that a required context is nil.
	ErrInvalidContext = errors.New("locker: invalid context")

	// ErrNilLock indicates that a required [Lock] is nil.
	ErrNilLock = errors.New("locker: nil lock")

	// ErrNilHandler indicates that a required [Handler] is nil.
	ErrNilHandler = errors.New("locker: nil handler")
)

// Locker creates named locks backed by a locking implementation.
type Locker interface {
	Lock(name string, ttl time.Duration) Lock
}

// Lock represents a named lock that may be acquired.
type Lock interface {
	// TryAcquire performs one acquisition attempt. It returns [ErrNotAcquired]
	// when another owner currently holds the lock. A nil ctx returns
	// [ErrInvalidContext]. A successful call returns a non-nil [Lease].
	TryAcquire(ctx context.Context) (Lease, error)

	// Acquire waits until the lock is acquired or ctx is done. A nil ctx
	// returns [ErrInvalidContext]. A successful call returns a non-nil [Lease].
	Acquire(ctx context.Context) (Lease, error)
}

// Lease represents ownership returned by a successful lock acquisition.
type Lease interface {
	// Release releases the lease if it is still owned by the caller. A nil ctx
	// returns [ErrInvalidContext].
	Release(ctx context.Context) error
}

// TransferableLease is a lease whose ownership token can be transferred to
// another process.
type TransferableLease interface {
	Lease

	// Token returns the ownership token for this lease.
	Token() string
}

// RestorableLock is a lock that can restore a lease from a transferred token.
type RestorableLock interface {
	Lock

	// Restore creates a lease for token without acquiring the lock again.
	Restore(token string) (Lease, error)
}

// RenewableLease is a lease that supports explicit renewal.
type RenewableLease interface {
	Lease

	// Refresh resets the lease expiration to ttl from the time the refresh
	// succeeds. A nil ctx returns [ErrInvalidContext].
	Refresh(ctx context.Context, ttl time.Duration) error
}

// NoopLocker creates no-op locks.
type NoopLocker struct{}

var _ Locker = NoopLocker{}

// Lock returns a no-op lock with the supplied name and ttl.
func (NoopLocker) Lock(name string, ttl time.Duration) Lock {
	return NoopLock{name: name, ttl: ttl}
}

// NoopLock is a no-op [Lock].
type NoopLock struct {
	name string
	ttl  time.Duration
}

var _ Lock = NoopLock{}

// TryAcquire validates the lock and returns a no-op lease.
func (l NoopLock) TryAcquire(ctx context.Context) (Lease, error) {
	if err := validate(ctx, l.name, l.ttl); err != nil {
		return nil, err
	}
	return NoopLease{}, nil
}

// Acquire validates the lock and returns a no-op lease.
func (l NoopLock) Acquire(ctx context.Context) (Lease, error) {
	return l.TryAcquire(ctx)
}

// NoopLease is a no-op [Lease].
type NoopLease struct{}

var _ Lease = NoopLease{}

// Release performs no work.
func (NoopLease) Release(ctx context.Context) error {
	return validateContext(ctx)
}

func validate(ctx context.Context, name string, ttl time.Duration) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if name == "" {
		return ErrInvalidName
	}
	if ttl <= 0 {
		return ErrInvalidTTL
	}
	return nil
}

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return ErrInvalidContext
	}
	return ctx.Err()
}
