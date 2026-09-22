package cache

import (
	"context"
	"errors"
	"time"
)

// ErrWriteUnconfirmed indicates that Put returned false without an error.
// The backend did not confirm a successful write; the value may not be cached.
var ErrWriteUnconfirmed = errors.New("cache: write was not confirmed")

// ReadWriter is the cache access required by a TypedView. Existing Store and
// Repository implementations satisfy it without adapters.
type ReadWriter interface {
	// Get decodes the cached value into dest. A miss leaves dest unchanged and
	// returns an error matching ErrNotFound. Other errors are backend errors.
	Get(ctx context.Context, key string, dest any) error
	// Put stores value with ttl, following the backend's encoding and TTL rules.
	// The bool reports a successful write. A non-nil error means the write
	// could not be completed or confirmed, regardless of the bool.
	Put(ctx context.Context, key string, value any, ttl time.Duration) (bool, error)
}

// TypedView binds cache reads, writes and cache-aside loading to a value type.
// It retains the supplied backend without owning or closing it, and introduces
// no key prefix or serialization rules. Concurrent use requires a backend that
// supports concurrent use. Construct a view with Typed; its zero value is unusable.
type TypedView[T any] struct {
	store ReadWriter
}

// Typed creates a reusable view of store for values of type T. Store must be
// non-nil. It can be a Repository, a Store or a backend implementing only Get/Put.
func Typed[T any](store ReadWriter) *TypedView[T] {
	return &TypedView[T]{store: store}
}

// Get retrieves a value of type T. A cache miss returns T's zero value and an
// error matching ErrNotFound. Backend errors are preserved; if decoding partially
// populated the value before failing, that value is returned alongside the error.
func (v *TypedView[T]) Get(ctx context.Context, key string) (T, error) {
	var value T
	err := v.store.Get(ctx, key, &value)
	return value, err
}

// Set stores value with ttl. It preserves backend errors and returns
// ErrWriteUnconfirmed if Put returns false, nil. A nil error means the backend
// reported success; durability and expiration guarantees remain backend-specific.
func (v *TypedView[T]) Set(ctx context.Context, key string, value T, ttl time.Duration) error {
	ok, err := v.store.Put(ctx, key, value, ttl)
	if err != nil {
		return err
	}
	if !ok {
		return ErrWriteUnconfirmed
	}
	return nil
}

// Remember retrieves a value, or calls load and caches its result on a miss.
// The same ctx is passed to the backend and load. Load must be non-nil when the
// key is missing. Concurrent misses may call load more than once.
//
// Only errors matching ErrNotFound via errors.Is invoke load. Other read errors
// return immediately. On a loader error, its value and error are
// returned without writing. On a write error, including ErrWriteUnconfirmed,
// the loaded value is returned with the error. A nil error means a cache hit
// or a successful load and backend-confirmed write.
func (v *TypedView[T]) Remember(ctx context.Context, key string, ttl time.Duration, load func(context.Context) (T, error)) (T, error) {
	value, err := v.Get(ctx, key)
	if !errors.Is(err, ErrNotFound) {
		return value, err
	}

	value, err = load(ctx)
	if err != nil {
		return value, err
	}
	return value, v.Set(ctx, key, value, ttl)
}
