# Repository Guidelines

## Project Structure & Module Organization
This is a multi-module Go repository. The root module is `github.com/go-fries/fries/v3` in `./go.mod`, and many component directories have their own `go.mod` files. Common packages live in top-level folders such as `cache/`, `codec/`, `env/`, `http/`, `redis/`, `mysql/`, `otel/`, `kratos/`, `gin/`, and `chi/`. Runnable samples and integration demos are under `examples/`. Protobuf contracts live in `contract/`, with root Buf configuration in `buf.yaml` and `buf.gen.yaml`. Developer tools are pinned in `internal/tools/`.

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
Recent history follows Conventional Commits, often with scopes, such as `fix(deps): update module ...` and `chore(deps): update ...`. Keep subjects imperative and scoped when useful. Pull requests should include a concise summary, linked issue when applicable, and test evidence listing commands run. For contract or generated-code changes, mention the Buf command used and include generated files in the same PR.
