# Repository Guidelines

## Project Structure & Module Organization
This is a multi-module Go repository. The root module lives at `./go.mod`, and many top-level component folders contain their own `go.mod` files. Common areas include:
- Component packages such as `cache/`, `codec/`, `env/`, `http/`, `redis/`, `mysql/`, `otel/`, `kratos/`, `gin/`, and `chi/`.
- `examples/` for runnable samples and integration demos.
- `internal/tools/` for pinned developer tooling used by the Makefile.
- `contract/` plus `buf.yaml`/`buf.gen.yaml` for protobuf contracts and generation.

## Build, Test, and Development Commands
Run these from the repository root:
- `make build` to compile all modules (`go build ./...` per module).
- `make test` for the default test suite across modules.
- `make test-short` for faster, `-short`-mode tests.
- `make test-race` to enable the race detector.
- `make test-coverage` to generate merged coverage (`coverage.txt`).
- `make lint` to run `go mod tidy` and `golangci-lint`.
- `make buf-lint` / `make buf-generate` for protobuf linting and codegen.

## Coding Style & Naming Conventions
- Formatting follows Go conventions (tabs for indentation); `golangci-lint` enables `gofumpt` and `goimports`.
- Package names are short, lowercase; exported identifiers use `CamelCase`.
- Tests use `*_test.go` files and `TestXxx`, `BenchmarkXxx`, `FuzzXxx` naming.

## Testing Guidelines
- Use `go test` via `make test` or per-module (e.g., `cd redis && go test ./...`).
- Race-sensitive changes should run `make test-race`.
- Long-running tests should honor `-short`.
- `github.com/stretchr/testify` is available for assertions when helpful.

## Commit & Pull Request Guidelines
- Recent history follows Conventional Commits, e.g. `fix(deps): update ...` or `chore(deps): ...` (often with PR numbers).
- Keep commit subjects imperative and scoped when relevant.
- PRs should include: a concise summary, linked issue (if any), and test evidence (commands run).
- Ensure `make lint` and `make test` are clean before opening a PR.
