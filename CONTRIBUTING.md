# Contributing

This document defines contribution conventions for this repository. It applies to new components, component refactors, public API design, and tests across all modules in this repository.

The goal is to keep components consistent, maintainable, and predictable for both users and maintainers.

## Repository Structure

This is a multi-module Go repository. The root module is `github.com/go-fries/fries/v3`, and many component directories have their own `go.mod` files.

Keep module boundaries explicit:

- update the nearest `go.mod` for the module being changed
- avoid unnecessary dependencies across components
- keep framework integrations under the framework or component they adapt

Framework integrations should follow the existing package layout. Kratos integrations live under `kratos/`, Hyperf Jet integrations live under `hyperf/jet/`, GORM integrations live under `gorm/`, and OpenTelemetry components live under `otel/` or under the integration package that emits telemetry.

## Design Principles

- Keep component boundaries clear. A component should own its configuration, runtime behavior, and tests within its package or module boundary.
- Prefer explicit behavior over implicit side effects. Errors should be returned where they occur and should not be hidden until later runtime behavior.
- Keep public APIs small and stable. Add exported identifiers only when they describe a real user-facing capability.
- Prefer established repository patterns over new local styles unless the component has a clear reason to differ.
- Avoid over-design. Add abstractions only when they reduce real complexity, remove meaningful duplication, or make public behavior easier to understand.

## Public Modules

New releasable component modules should update repository release and reporting metadata in the same change.

For public modules:

- add the module path to `versions.yaml`
- add the matching Codecov path rewrite to `codecov.yml`
- include package documentation in `doc.go`
- include a `README.md` when the module is intended for direct use
- add Go doc comments for exported identifiers
- provide a module-local `Version()` helper when the component exposes or reports version metadata

Public modules should be usable from Go documentation alone. README files should focus on installation, common usage, and behavior that is not obvious from type signatures.

## Component Configuration

Configuration state and option parsing should live in a package-local `config.go` when a component has configurable behavior.

Use `config.go` for:

- the internal `config` type
- default values
- the public `Option` contract
- the private `optionFunc` implementation
- public `WithXxx(...)` option helpers
- `newConfig(...)`
- small config-owned factory helpers such as `newLogger(...)`, `newResource(...)`, or `newTracerProvider(...)`

Runtime types should focus on lifecycle and behavior. A `Client`, `Logger`, `Middleware`, or similar type should generally call into its `config` for configured dependencies instead of carrying many option fields or scattered defaulting logic directly.

Config-owned factory helpers are appropriate when the created value is determined by configuration. Keep business logic, request handling, and runtime orchestration outside `config.go`.

## Options

Use an interface-based option contract with an unexported apply method by default:

```go
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}
```

Public option helpers should use `WithXxx(...) Option` naming and return `optionFunc`:

```go
func WithServiceName(name string) Option {
	return optionFunc(func(c *config) {
		c.serviceName = name
	})
}
```

Place option definitions in this order:

1. `Option`
2. `optionFunc`
3. `optionFunc.apply`
4. public `WithXxx(...)` helpers
5. `newConfig(...)`
6. config helper methods

Prefer this form over exporting or aliasing a raw `func(*config)` option type. It keeps option construction controlled inside the package while leaving room for future specialized option implementations.

When option application has a real failure path, use an error-returning option contract:

```go
type Option interface {
	apply(*config) error
}

type optionFunc func(*config) error

func (f optionFunc) apply(c *config) error {
	return f(c)
}
```

Use the non-error form for simple assignment and defaulting. Use the error-returning form when option-time validation or setup can fail. Return validation errors explicitly instead of deferring them to later runtime behavior.

## Testing

Tests should live beside the package under test and follow Go's standard `testing` package conventions.

Use `github.com/stretchr/testify` when it improves readability. This is encouraged for grouped assertions, error checks, and setup requirements, but it is not required for trivial standard-library checks.

Use `require` for conditions that must stop the current test before continuing, including:

- constructor errors
- setup failures
- nil checks before dereferencing
- preconditions for later assertions

Use `assert` for independent value checks where the test can continue and report multiple failures in one run.

Prefer explicit assertions that describe intent:

- `require.NoError`
- `require.ErrorIs`
- `assert.ErrorIs`
- `assert.Equal`
- `assert.Same`
- `assert.Contains`

Use table tests when they make behavior easier to scan. Keep each case focused on one observable behavior, and avoid broad table tests that hide setup complexity or make failures hard to diagnose.

## Validation

Run commands from the repository root unless working in a specific module.

Use repository-level commands for broad changes:

- `make build`
- `make test`
- `make test-short`
- `make test-race`
- `make test-coverage`
- `make lint`

Use module-specific Make targets for focused work, such as `make test/cache`, `make lint/cache`, and `make lint-fix/cache`.

Run `make lint` before submitting broad or public API changes. It runs `go mod tidy` across modules and then `golangci-lint`.

For protobuf changes, use the Buf targets defined by the repository:

- `make buf-lint`
- `make buf-build`
- `make buf-validate`
- `make buf-generate`
