# Signal Router

Route explicitly registered operating system signals to functions or handler
objects. The router owns signal subscriptions and handler workers; the
application owns exit policy and resource shutdown ordering.

## Installation

```bash
go get github.com/go-fries/fries/signal/v4
```

## Function handlers

Register handlers before calling `Run`. Registration reports invalid input
immediately; handler functions return errors from their operations.

```go
router := signal.New()
err := router.Listen(syscall.SIGHUP, func(ctx context.Context, _ os.Signal) error {
    return configuration.Reload(ctx)
})
if err != nil {
    return err
}
```

This fragment assumes an existing context-aware `configuration.Reload` operation.
The router logs handler errors and continues serving; it does not automatically
retry failed operations or return their errors from `Run`.

## Handler objects

Use an object when a handler owns dependencies or state. Its subscriptions are
configured at registration, not through a method on the handler.

```go
type ReloadHandler struct {
    Reload func(context.Context) error
}

func (h *ReloadHandler) Handle(ctx context.Context, _ os.Signal) error {
    return h.Reload(ctx)
}

// Fragment: configuration and router are supplied by application startup.
err := router.Register(
    &ReloadHandler{Reload: configuration.Reload},
    syscall.SIGHUP,
    syscall.SIGUSR1,
)
```

Handle the registration error before starting the router. `HandlerFunc` is also
available where an API requires a `Handler` object.

## Run and shut down

This complete Unix application reloads on SIGHUP and shuts down on SIGINT or
SIGTERM. Send `kill -HUP <pid>` to invoke its handler, then Ctrl-C to exit.

```go
package main

import (
    "context"
    "log/slog"
    "os"
    ossignal "os/signal"
    "syscall"
    "time"

    "github.com/go-fries/fries/signal/v4"
)

func main() {
    router := signal.New()
    if err := router.Listen(syscall.SIGHUP, func(ctx context.Context, _ os.Signal) error {
        slog.InfoContext(ctx, "reload requested")
        return nil // Replace with a context-aware reload operation.
    }); err != nil {
        slog.Error("registration failed", "error", err)
        return
    }

    exitCtx, stopExit := ossignal.NotifyContext(
        context.Background(), os.Interrupt, syscall.SIGTERM,
    )
    defer stopExit()

    // Do not derive the handler context from exitCtx: that would cancel active
    // handlers before graceful shutdown can wait for them.
    runCtx, cancelRun := context.WithCancel(context.Background())
    defer cancelRun()
    done := make(chan error, 1)
    go func() { done <- router.Run(runCtx) }()

    <-exitCtx.Done()
    stopExit()
    shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancelShutdown()
    if err := router.Shutdown(shutdownCtx); err != nil {
        slog.Error("shutdown failed", "error", err)
    }
    if err := <-done; err != nil {
        slog.Error("router failed", "error", err)
    }
}
```

The caller joins `Run` after requesting shutdown. `Shutdown` can bound the time
its caller waits, but cannot forcibly terminate a handler that ignores context
cancellation. If that happens, `Run` remains blocked until the handler exits.
The application's lifecycle coordinator must decide what to do after a timeout.
Calling `stopExit` restores exit-signal behavior only when no other subscriber
still handles those signals; a subsequent interrupt can then terminate the
process.

For applications that only need an exit-triggered context, use the standard
library's `os/signal.NotifyContext` directly; a router is unnecessary.

## Execution and delivery guarantees

- One worker runs each registration serially. A registration spanning multiple
  signals shares that worker and its pending slot.
- Different registrations run independently and may execute concurrently. There
  is no completion-order guarantee between registrations.
- Each registration stores at most one pending notification in addition to its
  active call. A full pending slot drops the **new** notification and logs a
  warning with its signal and registration number.
- Repeated `Register` or `Listen` calls create separate registrations, even for
  the same handler. If the same object is registered twice, it must support
  concurrent calls. Duplicate signals within one `Register` call are collapsed.
- The OS and standard library can also lose or coalesce signals. Those losses
  are not necessarily observable by the router. A signal is a notification, not
  a durable task; do not use the number of received signals as an operation count.
- Handler panics are recovered and logged with a stack trace. That registration
  remains available for later notifications.
- Runtime logs use `slog.Default()` unless `WithLogger` supplies a logger. Logging
  handlers must support concurrent calls and should not block reception.

## Lifecycle contract

| Operation or condition | Result |
| --- | --- |
| `New` | Creates an unstarted router; subscribes to nothing. |
| `Listen` / `Register` | Allowed only before `Run` or `Shutdown`; invalid calls add no registration. |
| `Run(ctx)` | Freezes registration, installs subscriptions, and blocks until all workers exit. |
| No registrations | `Run` waits without calling `os/signal.Notify` with an empty signal list. |
| Already-canceled Run context | Installs no subscriptions and returns the context error; the instance is consumed. |
| Run context cancellation | Cancels active handler contexts and discards pending work. |
| `Shutdown(ctx)` | Stops reception and new-call admission, discards pending work, and waits for active calls without canceling them. |
| Shutdown context cancellation | Cancels handler contexts and returns that caller's context error. The router remains stopping until workers exit. |
| Concurrent shutdown calls | Share cleanup. Any caller's canceled context can abort graceful waiting; the first abort error is retained. |
| Shutdown before Run | Permanently closes the instance without installing subscriptions. |
| Shutdown after completion | Returns the final Run result, even if the new caller's context is canceled. |

`Run` and completed `Shutdown` calls return the joined Run-context and first
shutdown-abort errors, or nil on graceful completion. Handler failures are logs,
not lifecycle results. Subscriptions are removed before waiting for workers;
a stuck handler does not retain the router's OS signal subscription.

Contexts must be non-nil. Do not call `Shutdown` from a handler: it would wait for
itself. A handler can cancel the application's context to request exit instead.
Create a new router to run again.

## Validation and signal ownership

- Nil functions and handlers, including typed nils, return `ErrInvalidHandler`.
- Empty signal lists, nil signals, typed nil signals, and non-comparable routing
  keys return `ErrInvalidSignal`.
- Registration after startup or shutdown returns `ErrRegistrationClosed`.
  Repeated `Run` calls return `ErrAlreadyStarted`.
- Use signals supported by `os/signal` on the target platform. The router does
  not validate platform signal numbers or make uncatchable signals such as
  SIGKILL and SIGSTOP catchable. Unix signals in these examples are not portable
  to every operating system.
- The router subscribes only to explicitly registered signals. It never calls
  `os.Exit`, installs default exit handlers, or calls global `signal.Reset` or
  `signal.Ignore`. Cleanup calls `signal.Stop` only for its own channel.

## Breaking changes

This redesign removes `Server`, `NewServer`, `WithHandlers`, `Start`, `Stop`,
`Handler.Listen`, and `AsyncHandler`. Use `New`, explicit `Listen` / `Register`,
`Run`, and `Shutdown`. `Handler.Handle` now returns `error`. Execution follows
the registration-based contract above rather than synchronous execution in the
signal receiver or an unbounded goroutine per asynchronous notification.
