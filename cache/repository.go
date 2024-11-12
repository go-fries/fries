package cache

import (
	"context"
	"time"
)

type Repository interface {
	Store
	Addable

	Missing(ctx context.Context, key string) (bool, error)
	Delete(ctx context.Context, key string) (bool, error)
	Set(ctx context.Context, key string, value any, ttl time.Duration) (bool, error)
}

type repository struct {
	Store
}

func NewRepository(store Store) Repository {
	return &repository{
		Store: store,
	}
}

func (r *repository) Missing(ctx context.Context, key string) (bool, error) {
	had, err := r.Store.Has(ctx, key)
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
