package cache

import (
	"context"
	"errors"
	"time"

	"github.com/go-kratos-ecosystem/components/v2/locker"
)

var (
	ErrNotFound   = errors.New("cache: the key is not found")
	ErrNotInteger = errors.New("cache: the key is not an integer")
)

type Store interface {
	Locker

	// Has returns true if the key exists in the cache.
	// If the key does not exist, the return value will be false, and the return error will be nil.
	// If the key exists, the return value will be true, and the return error will be nil.
	// otherwise, the return error will be the store error.
	Has(ctx context.Context, key string) (bool, error)

	// Get retrieves the value from the cache.
	// If the key does not exist, the dest will be unchanged, and the return error will be ErrNotFound.
	// If the key exists, the value will be unmarshaled to dest, and the return error will be nil.
	// otherwise, the return error will be the store error.
	Get(ctx context.Context, key string, dest any) error

	// Put stores the value into the cache with an expiration time.
	// If put success, the return value will be true, and the return error will be nil.
	// otherwise, the return value will be false, and the return error will be the store error.
	Put(ctx context.Context, key string, value any, ttl time.Duration) (bool, error)

	// Increment increments the value in the cache.
	// If the key does not exist, the before default value is 0.
	Increment(ctx context.Context, key string, value int) (int, error)

	// Decrement decrements the value in the cache.
	// If the key does not exist, the before default value is 0.
	Decrement(ctx context.Context, key string, value int) (int, error)

	Forever(ctx context.Context, key string, value any) (bool, error)

	Forget(ctx context.Context, key string) (bool, error)

	Flush(ctx context.Context) (bool, error)

	GetPrefix() string
}

type Addable interface {
	// Add stores the value into the cache with an expiration time if the key does not exist.
	// If the key exists, the return value will be false, and the return error will be nil.
	// If the key does not exist, the return value will be true, and the return error will be nil.
	// otherwise, the return error will be the store error.
	Add(ctx context.Context, key string, value any, ttl time.Duration) (bool, error)
}

type Locker interface {
	Lock(key string, ttl time.Duration) locker.Locker
}
