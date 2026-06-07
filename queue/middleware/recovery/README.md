# Queue Recovery Middleware

Panic recovery middleware for `github.com/go-fries/fries/queue/v3`.

## Installation

```bash
go get github.com/go-fries/fries/queue/middleware/recovery/v3
```

## Usage

```go
worker := queue.NewWorker(
	q,
	queue.Handle("send_email", handler),
	queue.WithMiddleware(recovery.New()),
)
```

Recovered panics are converted to handler errors so the worker can use its
configured retry or dead-letter policy.
