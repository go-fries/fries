# Poll

`poll` repeatedly observes a condition until it is satisfied, returns an error,
or its context is canceled. Checks run synchronously, and waits between checks
are context-aware.

## Installation

```bash
go get github.com/go-fries/fries/poll/v4
```

## Wait for a condition

The first check runs immediately. A positive interval is waited only after an
incomplete check:

```go
err := poll.Until(ctx, time.Second, func(ctx context.Context) (bool, error) {
	status, err := client.Status(ctx)
	if err != nil {
		return false, err
	}
	return status == "completed", nil
})
```

A condition error stops polling immediately. Use a context deadline to limit
the total polling lifetime:

```go
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()

err := poll.Until(ctx, time.Second, condition)
```

The condition itself must observe the supplied context while performing
blocking work. The package does not start a background goroutine or attempt to
forcibly stop a condition that ignores cancellation.

## Return the observed value

Use `UntilValue` when the final observed state is needed:

```go
job, err := poll.UntilValue(
	ctx,
	time.Second,
	func(ctx context.Context) (Job, bool, error) {
		job, err := client.Job(ctx, id)
		return job, job.Status == "completed", err
	},
)
```

On success, failure, or cancellation, `UntilValue` returns the most recent
value produced by the condition. It returns the type's zero value if the
condition has not run.

`poll` is for observing state until a condition becomes true. Use a retry
workflow instead when a transiently failing operation itself should be executed
again.
