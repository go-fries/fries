package cache

import (
	"context"
	"errors"
	"time"
)

// Get is a utility function that retrieves a value from the cache using the provided key.
// It unmarshals the value into the provided destination variable.
func Get[T any](ctx context.Context, repository Repository, key string) (T, error) {
	var result T
	return result, repository.Get(ctx, key, &result)
}

// Remember retrieves a value from the cache using the provided key.
// If the value is not found, it calls the callback, stores the result, and returns it.
// If the value is found in the cache, it returns the cached value and a nil error.
// Remember is not atomic; concurrent cache misses for the same key may call the callback more than once.
func Remember[T any](ctx context.Context, repository Repository, key string, ttl time.Duration, callback func() (T, error)) (T, error) {
	var result T
	err := repository.Get(ctx, key, &result)
	if !errors.Is(err, ErrNotFound) {
		return result, err
	}

	result, err = callback()
	if err != nil {
		return result, err
	}

	_, err = repository.Put(ctx, key, result, ttl)
	if err != nil {
		return result, err
	}

	return result, nil
}
