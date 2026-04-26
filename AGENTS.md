# Repository Guidelines

## Project Structure & Module Organization
This is a multi-module Go repository. The root module is `github.com/go-fries/fries/v3` in `./go.mod`, and many component directories have their own `go.mod` files. Common packages live in top-level folders such as `foundation/`, `event/`, `cache/`, `codec/`, `env/`, `http/`, `redis/`, `mysql/`, `otel/`, `kratos/`, `gin/`, and `chi/`. Runnable samples and integration demos are under `examples/`. Protobuf contracts live in `contract/`, with root Buf configuration in `buf.yaml` and `buf.gen.yaml`. Developer tools are pinned in `internal/tools/`.

Project working notes live under `.docs/`. Use `.docs/plans/` for task plans, phased checklists, and execution notes that coordinate longer work. The `.docs/plans/` directory is tracked, but individual plan files are local-only and should stay ignored. Use `.docs/memories/` for reusable project context that should survive across long-running tasks.

## Plans & Memories
When working on a long task, read relevant local files in `.docs/plans/` and tracked files in `.docs/memories/` before making assumptions. If you discover generally useful project information during execution, update `.docs/memories/` so future work can reuse it. Keep memories global and durable: prefer project conventions, module boundaries, dependency constraints, release practices, validation habits, and known architectural facts over one-off command logs or temporary status.

Files in `.docs/plans/` should use ordered, dated, status-bearing names: `NNN-YYYY-MM-DD-status-short-topic.md`. Use three-digit sequence numbers to preserve chronological order, and use status values such as `planned`, `in-progress`, `blocked`, or `completed`. When a plan status changes, rename the file so the filename status stays in sync with the checklist state.

## Build, Test, and Development Commands
Run commands from the repository root unless working in a specific module.

- `make tools` builds pinned local tools into `.tools/`.
- `make build` runs `go build ./...` for each repository module.
- `make test` runs the default test suite across modules.
- `make test-short` runs tests with `-short`.
- `make test-race` runs tests with the race detector.
- `make test-coverage` writes merged coverage to `coverage.txt`.
- `make lint` runs `go mod tidy` for all modules and `golangci-lint`.
- `make lint-fix` applies supported lint fixes.
- `make buf-lint`, `make buf-build`, `make buf-validate`, and `make buf-generate` validate or regenerate protobuf assets.

For focused work, use module-specific Make targets instead of manually changing directories. For example, `make test/cache` tests `cache/`, `make lint/cache` runs tidy plus lint, and `make lint-fix/cache` applies supported fixes.

## Coding Style & Naming Conventions
Follow standard Go formatting with tabs and idiomatic package layout. Package names should be short and lowercase; exported identifiers use `CamelCase`. The linter configuration enables formatting checks including `gofumpt` and `goimports`, so run `make lint` before submitting. Keep module boundaries clear: update the nearest `go.mod` and avoid introducing unnecessary dependencies across components.

## Testing Guidelines
Place tests beside the package under test using `*_test.go`. Use Go test naming conventions: `TestXxx`, `BenchmarkXxx`, and `FuzzXxx`. `github.com/stretchr/testify` is available when assertions improve readability. Long-running or integration-style tests should honor `testing.Short()`. Race-sensitive changes should be verified with `make test-race`.

## Commit & Pull Request Guidelines
Recent history follows Conventional Commits, often with scopes, such as `fix(deps): update module ...` and `chore(deps): update ...`. Keep subjects imperative and scoped when useful. Keep changes easy to review: each commit and pull request should address one coherent problem, and unrelated fixes, cleanup, or follow-up work should be split into separate branches or PRs. Pull requests should include a concise summary, linked issue when applicable, and test evidence listing commands run. For contract or generated-code changes, mention the Buf command used and include generated files in the same PR.
