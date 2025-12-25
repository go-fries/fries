# Queue

Queue component for Go Fries Components. Provides async task pushing and consumption with support for delayed jobs, priority queues, and automatic retries.

## Installation

```bash
go get github.com/go-fries/fries/queue/v3
go get github.com/go-fries/fries/queue/channel/v3  # In-memory driver
```

## Features

- **Driver-based architecture**: Pluggable backends (channel, Redis, RabbitMQ, etc.)
- **Priority queues**: Higher priority jobs are processed first
- **Delayed jobs**: Schedule jobs to run after a delay
- **Automatic retries**: Failed jobs are retried with exponential backoff
- **Dead letter queue**: Jobs that exceed max attempts are moved to a dead letter queue
- **Worker pool**: Configurable concurrency for job processing
- **Graceful shutdown**: Wait for in-flight jobs to complete

## Usage

### Basic Example

```go
package main

import (
    "context"
    "log"

    "github.com/go-fries/fries/queue/v3"
    "github.com/go-fries/fries/queue/channel/v3"
)

func main() {
    // Create channel driver (in-memory)
    driver := channel.New()

    // Create queue manager
    manager := queue.NewManager(driver)

    // Register handler for "emails" queue with 5 concurrent workers
    manager.Register("emails", queue.HandlerFunc(func(ctx context.Context, job *queue.Job) error {
        log.Printf("Processing job: %v", job.Payload())
        return nil
    }), queue.WithConcurrency(5))

    // Start the manager
    ctx := context.Background()
    manager.Start(ctx)

    // Push a job
    manager.PushOn(ctx, "emails", "send welcome email")

    // Graceful shutdown
    manager.Stop(ctx)
}
```

### Priority Jobs

```go
// Higher priority jobs are processed first
manager.Push(ctx, queue.NewJob("urgent task",
    queue.WithQueue("emails"),
    queue.WithPriority(100),
))

manager.Push(ctx, queue.NewJob("normal task",
    queue.WithQueue("emails"),
    queue.WithPriority(1),
))
```

### Delayed Jobs

```go
// Job will be available after 5 minutes
manager.Later(ctx, 5*time.Minute, queue.NewJob("delayed notification",
    queue.WithQueue("notifications"),
))

// Or use WithDelay option
manager.Push(ctx, queue.NewJob("scheduled task",
    queue.WithQueue("emails"),
    queue.WithDelay(10*time.Second),
))
```

### Retry Configuration

```go
// Job will be retried up to 3 times with exponential backoff
manager.Push(ctx, queue.NewJob("important task",
    queue.WithQueue("emails"),
    queue.WithMaxAttempts(3),
))
```

### Kratos Integration

```go
import (
    "github.com/go-kratos/kratos/v2"
    "github.com/go-fries/fries/queue/v3"
    "github.com/go-fries/fries/queue/channel/v3"
)

func main() {
    driver := channel.New()
    manager := queue.NewManager(driver)

    manager.Register("default", handler, queue.WithConcurrency(10))

    // Manager implements contract.Server interface
    app := kratos.New(
        kratos.Server(manager),
    )

    app.Run()
}
```

## Available Drivers

| Driver | Package | Description |
|--------|---------|-------------|
| Channel | `queue/channel/v3` | In-memory driver using Go channels and heap |
| Redis | `queue/redis/v3` | Redis-backed driver (coming soon) |
| AMQP | `queue/amqp/v3` | RabbitMQ driver (coming soon) |

## Driver Interface

Implement the `Driver` interface to create custom queue backends:

```go
type Driver interface {
    Push(ctx context.Context, job *Job) error
    Pop(ctx context.Context, queues ...string) (*Job, error)
    Ack(ctx context.Context, job *Job) error
    Fail(ctx context.Context, job *Job, err error) error
    Size(ctx context.Context, queue string) (int64, error)
    Dead(ctx context.Context, queue string) (int64, error)
    Clear(ctx context.Context, queue string) error
    ClearDead(ctx context.Context, queue string) error
}
```

Optional interfaces for additional features:

- `Startable` / `Stoppable`: Lifecycle management
- `Retryable`: Retry dead jobs
- `Inspectable`: Peek queue contents

## License

The MIT License (MIT). Please see [License File](../LICENSE) for more information.