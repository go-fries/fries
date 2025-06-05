package cached

import (
	"context"
	"errors"
	"time"
)

var defaultCacher Cacher = &noCacher{}

var ErrCacherKeyNotFound = errors.New("embedding/cached: key not found in cacher")

type Cacher interface {
	// Set stores the value in the cache with the given key.
	// If the key already exists, it will be overwritten.
	Set(ctx context.Context, key string, value []float64, expire time.Duration) error

	// Get retrieves the value from the cache with the given key.
	// If the key does not exist, the err will be ErrCacherKeyNotFound.
	// If the value is not of type []float64, it returns an error.
	Get(ctx context.Context, key string) ([]float64, error)
}

type noCacher struct{}

var _ Cacher = (*noCacher)(nil)

func (c *noCacher) Set(_ context.Context, _ string, _ []float64, _ time.Duration) error {
	return nil
}

func (c *noCacher) Get(_ context.Context, _ string) ([]float64, error) {
	return nil, ErrCacherKeyNotFound
}
