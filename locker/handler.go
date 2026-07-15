package locker

import (
	"context"
	"errors"
)

// Handler is synchronous work executed while a lock is held.
type Handler func(context.Context) error

// Do waits to acquire lock, executes handler, and releases the resulting
// lease. Handler and release errors are joined. A nil ctx, lock, or handler
// returns [ErrInvalidContext], [ErrNilLock], or [ErrNilHandler], respectively.
func Do(ctx context.Context, lock Lock, handler Handler) error {
	if err := validateHandler(ctx, lock, handler); err != nil {
		return err
	}

	lease, err := lock.Acquire(ctx)
	if err != nil {
		return err
	}

	return handle(ctx, lease, handler)
}

// Try performs one lock acquisition attempt. If successful, it executes
// handler and releases the resulting lease. Handler and release errors are
// joined. A nil ctx, lock, or handler returns [ErrInvalidContext], [ErrNilLock],
// or [ErrNilHandler], respectively.
func Try(ctx context.Context, lock Lock, handler Handler) error {
	if err := validateHandler(ctx, lock, handler); err != nil {
		return err
	}

	lease, err := lock.TryAcquire(ctx)
	if err != nil {
		return err
	}

	return handle(ctx, lease, handler)
}

func handle(ctx context.Context, lease Lease, handler Handler) (err error) {
	defer func() {
		releaseErr := lease.Release(context.WithoutCancel(ctx))
		switch {
		case err == nil:
			err = releaseErr
		case releaseErr != nil:
			err = errors.Join(err, releaseErr)
		}
	}()

	return handler(ctx)
}

func validateHandler(ctx context.Context, lock Lock, handler Handler) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if lock == nil {
		return ErrNilLock
	}
	if handler == nil {
		return ErrNilHandler
	}
	return nil
}
