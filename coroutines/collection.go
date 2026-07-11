package coroutines

import (
	"context"
	"fmt"
)

// ForEach calls fn for every value with at most limit calls running
// concurrently.
//
// The first callback error cancels the context passed to the other calls.
// Limit must be greater than zero.
func ForEach[T any](
	ctx context.Context,
	limit int,
	values []T,
	fn func(context.Context, T) error,
) error {
	if err := validateLimit(limit); err != nil {
		return err
	}
	if fn == nil {
		return fmt.Errorf("%w: ForEach", ErrNilFunc)
	}
	if len(values) == 0 {
		return nil
	}

	return execute(ctx, limit, len(values), func(ctx context.Context, index int) error {
		return fn(ctx, values[index])
	})
}

// Map calls fn for every value with at most limit calls running concurrently.
// Successful results preserve the order of values.
//
// The first callback error cancels the context passed to the other calls. On
// failure, Map returns a nil result slice and the error. Limit must be greater
// than zero.
func Map[T, R any](
	ctx context.Context,
	limit int,
	values []T,
	fn func(context.Context, T) (R, error),
) ([]R, error) {
	if err := validateLimit(limit); err != nil {
		return nil, err
	}
	if fn == nil {
		return nil, fmt.Errorf("%w: Map", ErrNilFunc)
	}

	results := make([]R, len(values))
	if len(values) == 0 {
		return results, nil
	}

	err := execute(ctx, limit, len(values), func(ctx context.Context, index int) error {
		result, err := fn(ctx, values[index])
		if err != nil {
			return err
		}

		results[index] = result

		return nil
	})
	if err != nil {
		return nil, err
	}

	return results, nil
}
