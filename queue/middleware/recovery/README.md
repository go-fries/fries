# Queue Recovery Middleware

Panic recovery middleware for `github.com/go-fries/fries/queue/v4`.

## Installation

```bash
go get github.com/go-fries/fries/queue/middleware/recovery/v4
```

## Usage

```go
package main

import (
	"github.com/go-fries/fries/queue/middleware/recovery/v4"
	"github.com/go-fries/fries/queue/v4"
)

func newWorker(q queue.Queue, handler queue.Handler) *queue.Worker {
	return queue.NewWorker(
		q,
		queue.Handle("send_email", handler),
		queue.WithMiddleware(recovery.New()),
	)
}
```

Recovered panics are converted to handler errors so the worker can use its
configured retry or dead-letter policy.

The default recovery handler logs task ID, type, queue, and attempt. It does not
log task payload or metadata. Use `WithHandler` when an application needs custom
panic reporting.
