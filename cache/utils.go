package cache

import (
	"context"
	"errors"
	"time"
)

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
