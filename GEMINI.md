# Fries (Go Component Toolkit)

## Project Overview

**Fries** (formerly `go-kratos-ecosystem/components`) is a modular collection of Go libraries and components designed to build robust applications. It follows a toolkit approach where developers can pick and choose specific packages (like `event`, `cache`, `filesystem`) or use the `foundation` package to structure an entire application using a Service Provider pattern.

*   **Language:** Go (>= 1.25.0)
*   **Architecture:** Modular, Component-based, Service Provider (DI) pattern.
*   **Key Pattern:** The `foundation.Kernel` manages application lifecycle via `Bootstrap` and `Terminate` methods defined in `Provider` interfaces.

## Key Directories & Components

*   **`foundation/`**: The core application kernel. Contains `Kernel`, `Provider`, and `Handler` interfaces for lifecycle management.
*   **`event/`**: A typed event dispatcher with support for middleware (e.g., panic recovery).
*   **`config/`**: A generic configuration propagation mechanism using `context.Context`.
*   **`cache/`, `redis/`, `mysql/`**: Data access and caching components.
*   **`chi/`, `gin/`**: Wrappers/Integrations for popular HTTP routers.
*   **`filesystem/`**: Abstraction for file storage (Local, S3, OSS).
*   **`crontab/`**: Cron job scheduling.
*   **`internal/`**: Internal shared utilities and tools.
*   **`examples/`**: Usage examples for various components.

## Development & Usage

This project uses a `Makefile` to manage the build, test, and lint workflows across its multiple modules.

### Prerequisites

*   Go >= 1.25.0
*   `make`

### Common Commands

*   **Install Tools:**
    ```bash
    make tools
    ```
    *Installs local build tools (lint, release, etc.) to `.tools/`.*

*   **Build All Modules:**
    ```bash
    make build
    ```

*   **Run Tests:**
    ```bash
    make test
    ```
    *Supports variants like `make test-race`, `make test-short`.*

*   **Linting:**
    ```bash
    make lint
    ```
    *Runs `go mod tidy` and `golangci-lint`.*

*   **Fix Lint Issues:**
    ```bash
    make golangci-lint-fix
    ```

### Core Concepts

#### Service Providers (`foundation`)
Applications structured with `fries` typically use a `Kernel` that registers `Providers`.
```go
type Provider interface {
    Bootstrap(context.Context) (context.Context, error) // Init
    Terminate(context.Context) (context.Context, error) // Cleanup
}
```

#### Event Dispatching (`event`)
The event system is typed and middleware-friendly.
```go
dispatcher := event.NewDispatcher()
dispatcher.Dispatch(ctx, &MyEvent{...})
```
