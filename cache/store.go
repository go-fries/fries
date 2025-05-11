package cache

import (
	"context"
	"errors"
	"time"

	"github.com/go-fries/fries/locker/v3"
)

var ErrNotFound = errors.New("cache: the key is not found")

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

	// Forever stores the key-value pair in the cache permanently until explicitly removed.
	// It returns true if the operation succeeds, or false with an error describing the failure.
	Forever(ctx context.Context, key string, value any) (bool, error)

	// Forget removes the specified key from the cache.
	// It returns true if the key was successfully deleted, or false with an error if the deletion fails.
	Forget(ctx context.Context, key string) (bool, error)

	// Flush clears all keys and values from the cache.
	// It returns true if the cache is successfully flushed, or false with an error if the operation fails.
	Flush(ctx context.Context) (bool, error)

	// GetPrefix returns the prefix string used for all cache keys managed by this store.
	// This prefix helps isolate cache entries between different applications or services.
	GetPrefix() string
}

type Addable interface {
	// Add stores the value into the cache with an expiration time if the key does not exist.
	// If the key exists, the return value will be false, and the return error will be nil.
	// If the key does not exist, the return value will be true, and the return error will be nil.
	// otherwise, the return error will be the store error.
	Add(ctx context.Context, key string, value any, ttl time.Duration) (bool, error)
}

// Locker interface defines methods for acquiring a distributed lock with a specified time-to-live (TTL).
// It abstracts the underlying locking mechanism and allows operations like cache-based locking.
type Locker interface {
	// Lock attempts to acquire a lock identified by the provided key and sets its TTL.
	// The returned locker.Locker can be used to perform further operations such as unlocking.
	// Parameters:
	//   key - A unique identifier for the lock.
	//   ttl - The time-to-live duration for the lock, after which it will automatically be released.
	// Return value:
	//   Returns a locker.Locker instance to manage the acquired lock.
	Lock(key string, ttl time.Duration) locker.Locker
}
