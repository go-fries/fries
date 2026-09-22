package cache

import (
	"context"
	"errors"
	"time"

	"github.com/go-fries/fries/locker/v4"
)

// ErrNotFound indicates that a requested cache key does not exist.
var ErrNotFound = errors.New("cache: the key is not found")

// Store is the base cache backend contract.
type Store interface {
	locker.Locker

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
	// The bool reports whether the backend reports a successful write. A false,
	// nil result is a negative status, not a confirmed successful write.
	// A non-nil error means the write could not be completed or confirmed;
	// callers must not infer from it that the key was left unchanged.
	Put(ctx context.Context, key string, value any, ttl time.Duration) (bool, error)

	// Increment increments the value in the cache.
	// If the key does not exist, the before default value is 0.
	Increment(ctx context.Context, key string, value int) (int, error)

	// Decrement decrements the value in the cache.
	// If the key does not exist, the before default value is 0.
	Decrement(ctx context.Context, key string, value int) (int, error)

	// Forever stores the key-value pair in the cache without an expiration time.
	// Its result and error have the same meaning as Put.
	Forever(ctx context.Context, key string, value any) (bool, error)

	// Forget removes the specified key from the cache.
	// It returns true, nil when deletion is reported and false, nil when the key
	// was absent. A non-nil error means deletion could not be completed or confirmed.
	Forget(ctx context.Context, key string) (bool, error)

	// Flush clears all keys and values from the cache.
	// Its bool reports whether the backend reports a successful clearing pass.
	// A successful pass does not guarantee atomic removal or that no concurrent
	// writer added keys. A non-nil error may follow partial removal.
	Flush(ctx context.Context) (bool, error)

	// GetPrefix returns the prefix string used for all cache keys managed by this store.
	// This prefix helps isolate cache entries between different applications or services.
	GetPrefix() string
}

// Addable is the existing conditional-write interface. It does not by itself
// guarantee atomicity: Repository may implement it using a check followed by a
// write. Consult the backend's Add contract when atomicity is required.
type Addable interface {
	// Add stores the value into the cache with an expiration time if the key does not exist.
	// It returns true, nil when insertion is reported and false, nil when the
	// key exists or the backend reports a negative write status.
	// A non-nil error means the result could not be completed or confirmed.
	Add(ctx context.Context, key string, value any, ttl time.Duration) (bool, error)
}
