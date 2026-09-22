package cache

import (
	"context"
	"time"
)

// Repository adds convenience operations to a Store. Its Add operation may use
// a non-atomic fallback; atomicity depends on the backend's Add implementation.
type Repository interface {
	Store
	Addable

	Missing(ctx context.Context, key string) (bool, error)
	// Delete is an alias for Forget and preserves its result and error.
	Delete(ctx context.Context, key string) (bool, error)
	// Set is an alias for Put and preserves its result and error.
	Set(ctx context.Context, key string, value any, ttl time.Duration) (bool, error)
}

type repository struct {
	Store
}

// NewRepository wraps store with convenience operations. It forwards Add to
// stores implementing [Addable], otherwise checking existence before writing.
// This check-then-write fallback is not atomic.
func NewRepository(store Store) Repository {
	return &repository{
		Store: store,
	}
}

func (r *repository) Missing(ctx context.Context, key string) (bool, error) {
	had, err := r.Has(ctx, key)
	if err != nil {
		return false, err
	}

	return !had, nil
}

func (r *repository) Add(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	// if the store is addable, use it
	if store, ok := r.Store.(Addable); ok {
		return store.Add(ctx, key, value, ttl)
	}

	// otherwise, use the default implementation
	if missing, err := r.Missing(ctx, key); err != nil {
		return false, err
	} else if missing {
		status, err := r.Set(ctx, key, value, ttl)
		if err != nil {
			return false, err
		}
		return status, nil
	}

	return false, nil
}

func (r *repository) Delete(ctx context.Context, key string) (bool, error) {
	return r.Forget(ctx, key)
}

func (r *repository) Set(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	return r.Put(ctx, key, value, ttl)
}
