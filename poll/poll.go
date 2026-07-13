package poll

import (
	"context"
	"time"
)

// Condition observes state and reports whether polling is complete.
// An error stops polling immediately.
type Condition func(context.Context) (done bool, err error)

// ValueCondition observes state and returns its latest value together with
// whether polling is complete. An error stops polling immediately.
type ValueCondition[T any] func(context.Context) (value T, done bool, err error)

// Until calls condition until it reports completion, returns an error, or ctx
// is canceled. The first call is immediate; interval is waited only between
// incomplete calls.
//
// The context must not be nil. Condition runs synchronously and should observe
// ctx while performing blocking work. Context cancellation that occurs during
// a condition call takes precedence over the error returned by that call.
// Until panics if condition is nil or interval is not positive.
func Until(ctx context.Context, interval time.Duration, condition Condition) error {
	if condition == nil {
		panic("poll: nil condition")
	}

	_, err := untilValue(ctx, interval, func(ctx context.Context) (struct{}, bool, error) {
		done, err := condition(ctx)
		return struct{}{}, done, err
	})
	return err
}

// UntilValue calls condition until it reports completion, returns an error, or
// ctx is canceled. The first call is immediate; interval is waited only between
// incomplete calls. It returns the most recent value produced by condition,
// including when polling fails or is canceled. If condition has not run, it
// returns the zero value of T.
//
// The context must not be nil. Condition runs synchronously and should observe
// ctx while performing blocking work. Context cancellation that occurs during
// a condition call takes precedence over the error returned by that call.
// UntilValue panics if condition is nil or interval is not positive.
func UntilValue[T any](
	ctx context.Context,
	interval time.Duration,
	condition ValueCondition[T],
) (T, error) {
	if condition == nil {
		panic("poll: nil condition")
	}

	return untilValue(ctx, interval, condition)
}

func untilValue[T any](
	ctx context.Context,
	interval time.Duration,
	condition func(context.Context) (T, bool, error),
) (T, error) {
	if interval <= 0 {
		panic("poll: interval must be greater than zero")
	}

	var value T
	for {
		if err := ctx.Err(); err != nil {
			return value, err
		}

		var done bool
		var err error
		value, done, err = condition(ctx)
		// Prefer cancellation that occurred during the condition call over its
		// returned result or error.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return value, ctxErr
		}
		if err != nil {
			return value, err
		}
		if done {
			return value, nil
		}
		if err := wait(ctx, interval); err != nil {
			return value, err
		}
	}
}

func wait(ctx context.Context, interval time.Duration) error {
	timer := time.NewTimer(interval)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
