package parallel

import (
	"context"
	"fmt"
)

// Result contains the outcome of processing one input value.
type Result[T any] struct {
	// Value is the callback result. It is the zero value of T when the callback
	// fails before producing a value.
	Value T
	// Err is the callback error for the corresponding input value.
	Err error
}

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

// MapResults calls fn for every value with at most limit calls running
// concurrently and returns one Result for each input value.
//
// Unlike Map, an error returned by fn is stored in the corresponding Result
// and does not cancel other callbacks. Results preserve the order of values.
// If ctx is canceled, MapResults returns the partial results and the context
// error; values whose callbacks did not start receive that error in Result.Err.
// Limit must be greater than zero.
func MapResults[T, R any](
	ctx context.Context,
	limit int,
	values []T,
	fn func(context.Context, T) (R, error),
) ([]Result[R], error) {
	if err := validateLimit(limit); err != nil {
		return nil, err
	}
	if fn == nil {
		return nil, fmt.Errorf("%w: MapResults", ErrNilFunc)
	}

	results := make([]Result[R], len(values))
	if len(values) == 0 {
		return results, nil
	}

	completed := make([]bool, len(values))
	err := execute(ctx, limit, len(values), func(ctx context.Context, index int) error {
		results[index].Value, results[index].Err = fn(ctx, values[index])
		completed[index] = true

		return nil
	})
	if err != nil {
		for index := range results {
			if !completed[index] {
				results[index].Err = err
			}
		}
	}

	return results, err
}

// Filter calls fn for every value with at most limit calls running
// concurrently and returns the values for which fn returns true.
//
// The returned values preserve their input order. The first callback error
// cancels the context passed to the other calls. On failure, Filter returns a
// nil result slice and the error. Limit must be greater than zero.
func Filter[T any](
	ctx context.Context,
	limit int,
	values []T,
	fn func(context.Context, T) (bool, error),
) ([]T, error) {
	if err := validateLimit(limit); err != nil {
		return nil, err
	}
	if fn == nil {
		return nil, fmt.Errorf("%w: Filter", ErrNilFunc)
	}

	matches, err := Map(ctx, limit, values, fn)
	if err != nil {
		return nil, err
	}

	filtered := make([]T, 0, len(values))
	for index, match := range matches {
		if match {
			filtered = append(filtered, values[index])
		}
	}

	return filtered, nil
}
