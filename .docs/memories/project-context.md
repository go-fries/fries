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

## Validation Habits

- Run commands from the repository root unless focusing on a specific module.
- Use `make build` for repository-wide builds across modules.
- Use `make test`, `make test-short`, `make test-race`, and
  `make test-coverage` for repository-level test workflows.
- Use module-specific Make targets for focused work, such as `make test/cache`,
  `make lint/cache`, and `make lint-fix/cache`.
- Run `make lint` before submitting broader changes because it runs `go mod tidy`
  across modules and then `golangci-lint`.

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

## Cache Component Notes

- A focused cache-component review plan currently lives in
  `.docs/plans/001-2026-04-26-planned-cache-component-optimization.md`.
- The review scope is intentionally limited to `cache/` and `cache/redis/`.
- Initial local verification passed for `cache/` with `go test ./...` and
  `go test -race ./...`.
- Initial `cache/redis` tests require local Redis at `:6379`; in the current
  restricted environment they failed with `operation not permitted`.
