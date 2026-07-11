package coroutines

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"
)

// Task is a unit of concurrent work.
//
// A task should stop promptly when ctx is canceled. Tasks must not retain ctx
// after returning.
type Task func(ctx context.Context) error

// Run executes tasks concurrently and waits for them to finish.
//
// The first task error cancels the context passed to the other tasks. Run
// returns that error after all started tasks have returned. Use RunLimit when
// the number of concurrently running tasks must be bounded.
func Run(ctx context.Context, tasks ...Task) error {
	if err := validateTasks(tasks); err != nil {
		return err
	}
	if len(tasks) == 0 {
		return nil
	}

	return execute(ctx, len(tasks), len(tasks), func(ctx context.Context, index int) error {
		return tasks[index](ctx)
	})
}

// RunLimit executes tasks with at most limit tasks running concurrently and
// waits for them to finish.
//
// The first task error cancels the context passed to the other tasks. Limit
// must be greater than zero.
func RunLimit(ctx context.Context, limit int, tasks ...Task) error {
	if err := validateLimit(limit); err != nil {
		return err
	}
	if err := validateTasks(tasks); err != nil {
		return err
	}
	if len(tasks) == 0 {
		return nil
	}

	return execute(ctx, limit, len(tasks), func(ctx context.Context, index int) error {
		return tasks[index](ctx)
	})
}

func execute(
	ctx context.Context,
	limit int,
	count int,
	fn func(context.Context, int) error,
) error {
	group, groupContext := errgroup.WithContext(ctx)
	group.SetLimit(limit)

	for index := range count {
		group.Go(func() error {
			if err := context.Cause(groupContext); err != nil {
				return err
			}

			return fn(groupContext, index)
		})
	}

	if err := group.Wait(); err != nil {
		return err
	}

	return context.Cause(ctx)
}

func validateLimit(limit int) error {
	if limit <= 0 {
		return fmt.Errorf("%w: got %d", ErrInvalidLimit, limit)
	}

	return nil
}

func validateTasks(tasks []Task) error {
	for index, task := range tasks {
		if task == nil {
			return fmt.Errorf("%w: index %d", ErrNilTask, index)
		}
	}

	return nil
}
