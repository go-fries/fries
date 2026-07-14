# Lifecycle

Lifecycle coordinates application startup, execution, and graceful shutdown.

## Installation

```bash
go get github.com/go-fries/fries/lifecycle/v4
```

## Usage

```go
runner := lifecycle.New(
	lifecycle.WithProviders(
		configProvider,
		eventProvider,
		telemetryProvider,
	),
	lifecycle.WithShutdownTimeout(10*time.Second),
)

err := runner.Run(ctx, func(ctx context.Context) error {
	return application.Run(ctx)
})
```

Providers bootstrap in registration order. Each provider may return a derived
context for the next provider and the application handler. Providers shut down
in reverse order and may likewise pass a derived context to the next shutdown
step.

If startup fails, only providers that started successfully are shut down.
Shutdown continues after individual failures, and the runner joins application
and shutdown errors so callers can inspect them with `errors.Is` and
`errors.As`.

The runner creates a dedicated shutdown context that preserves values added
during bootstrap without inheriting cancellation from the runtime context.
Providers must still observe that context themselves; a timeout cannot forcibly
stop a provider that ignores cancellation.

A nil context is treated as `context.Background()`. A nil handler is treated as
a no-op: providers bootstrap and then immediately shut down.
