package ratelimit

import (
	"context"
	"fmt"
)

// Limiter applies one [Limit] through a [Store].
//
// A Limiter is safe for concurrent use when its Store is safe for concurrent
// use.
type Limiter struct {
	store Store
	limit Limit
}

// New creates a [Limiter] that applies limit through store.
//
// New returns [ErrNilStore] when store is nil and [ErrInvalidLimit] when limit
// is invalid.
func New(store Store, limit Limit) (*Limiter, error) {
	if store == nil {
		return nil, ErrNilStore
	}
	if err := validateLimit(limit); err != nil {
		return nil, err
	}
	return &Limiter{store: store, limit: limit}, nil
}

// Allow attempts to consume one unit for key.
//
// A nil ctx returns [ErrInvalidContext], and an empty key returns
// [ErrInvalidKey]. A rejected request returns a decision whose Allowed field is
// false and a nil error.
func (l *Limiter) Allow(ctx context.Context, key string) (Decision, error) {
	return l.AllowN(ctx, key, 1)
}

// AllowN attempts to consume cost units for key atomically.
//
// Cost must be between one and the configured burst. A nil ctx returns
// [ErrInvalidContext], and an empty key returns [ErrInvalidKey]. A rejected
// request returns a decision whose Allowed field is false and a nil error.
func (l *Limiter) AllowN(
	ctx context.Context,
	key string,
	cost int,
) (Decision, error) {
	if err := validateContext(ctx); err != nil {
		return Decision{}, err
	}
	if key == "" {
		return Decision{}, ErrInvalidKey
	}
	if cost < 1 || cost > l.limit.Burst {
		return Decision{}, ErrInvalidCost
	}

	decision, err := l.store.Take(ctx, TakeRequest{
		Key:   key,
		Limit: l.limit,
		Cost:  cost,
	})
	if err != nil {
		return Decision{}, fmt.Errorf("ratelimit: take %q: %w", key, err)
	}
	decision.Limit = l.limit
	return decision, nil
}

// Reset removes the stored rate-limit state for key.
//
// A nil ctx returns [ErrInvalidContext], and an empty key returns
// [ErrInvalidKey].
func (l *Limiter) Reset(ctx context.Context, key string) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	if key == "" {
		return ErrInvalidKey
	}
	if err := l.store.Reset(ctx, key); err != nil {
		return fmt.Errorf("ratelimit: reset %q: %w", key, err)
	}
	return nil
}

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return ErrInvalidContext
	}
	return context.Cause(ctx)
}
