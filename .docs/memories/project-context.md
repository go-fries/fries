# Project Context Memory

## Repository Identity

- Repository: `go-fries/fries`.
- Project type: multi-module Go repository.
- Root module: `github.com/go-fries/fries/v3`.
- Default local branch observed: `3.x`.
- Root Go version in `go.mod`: `1.25.0`.

## Module Layout

- The repository contains many independent Go modules, each with its own
  `go.mod`.
- Common component modules include `foundation/`, `event/`, `cache/`, `codec/`,
  `env/`, `http/`, `redis/`, `mysql/`, `otel/`, `gin/`, `chi/`, and `locker/`.
- Protobuf contracts live in `contract/`, with root Buf configuration in
  `buf.yaml` and `buf.gen.yaml`.
- Developer tools are pinned under `internal/tools/` and built into `.tools/`.
- Integration packages are normally placed under the framework or component they
  adapt. For example, Kratos integrations live under `kratos/`, Hyperf Jet
  integrations live under `hyperf/jet/`, and GORM integrations live under
  `gorm/`.
- The `queue/` component owns durable task queue primitives, including task
  envelopes, producers, workers, retry policies, middleware, and the in-memory
  queue. Queue adapters should live under the component, such as
  `queue/redis/` for Redis Streams.
- Queue implementations stay byte-oriented through `Task.Payload []byte`. Typed payload
  helpers such as `TaskFor[T]`, `EnqueueFor`, and `HandleFor` live in the core
  module as a convenience layer and should not change queue contracts.
- Queue task metadata is `Task.Metadata map[string]string`; it is task-level
  application metadata, not queue delivery state. Delivery-specific values stay
  in `Lease` implementations.
- Queue middleware packages live under `queue/middleware/`; for example,
  `queue/middleware/recovery/` provides panic recovery middleware that converts
  panics into handler errors for retry or dead-letter handling.
- Queue adapter packages live under `queue/adapter/`; for example,
  `queue/adapter/redis/` adapts Redis Streams to the `queue.Queue` interface.

## Public Module Conventions

- New releasable modules should be added to `versions.yaml` and should have a
  matching Codecov path rewrite in `codecov.yml`.
- Public modules should include package documentation in `doc.go`, user-facing
  usage notes in `README.md` when the module is intended for direct use, and Go
  doc comments for exported identifiers.
- Versioned component modules commonly expose a module-local `Version()` helper
  returning the current repository release version. Use that pattern when adding
  modules that report instrumentation or component version metadata.

## OpenTelemetry Conventions

- OpenTelemetry integration package names should be short and match the final
  path segment, such as `otel` for modules whose import path ends in `/otel/v3`.
- Instrumentation scope names should remain stable and match the full module
  import path. Scope versions should come from the module's `Version()` helper
  unless the caller explicitly overrides them.
- Prefer official OpenTelemetry semantic convention constants and helper
  functions before introducing raw or package-specific attribute names. Use
  package-specific attributes only when no suitable semantic convention exists.

## Validation Habits

- Run commands from the repository root unless focusing on a specific module.
- Use `make build` for repository-wide builds across modules.
- Use `make test`, `make test-short`, `make test-race`, and
  `make test-coverage` for repository-level test workflows.
- Use module-specific Make targets for focused work, such as `make test/cache`,
  `make lint/cache`, and `make lint-fix/cache`.
- Run `make lint` before submitting broader changes because it runs `go mod tidy`
  across modules and then `golangci-lint`.
- In restricted local environments, Go and golangci-lint cache writes may need
  explicit writable cache directories. Prefer setting `GOCACHE` and
  `GOLANGCI_LINT_CACHE` to environment-appropriate temporary directories rather
  than relying on user-level cache paths.

## Local Planning Conventions

- Files in `.docs/plans/` should use ordered, dated, status-bearing names:
  `NNN-YYYY-MM-DD-status-short-topic.md`.
- Use three-digit sequence numbers to preserve chronological order, such as
  `001`, `002`, and `003`.
- Use status values that describe the current execution state:
  `planned`, `in-progress`, `blocked`, or `completed`.
- When a plan status changes, rename the file to keep the filename status in
  sync with the checklist state.
- The `.docs/plans/` directory is tracked through `.docs/plans/.gitkeep`, but
  individual plan markdown files are local-only and ignored.
- `.docs/memories/` is tracked and should hold durable project context useful
  across future long-running tasks.
