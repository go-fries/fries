# Contributing

This document defines contribution conventions for this repository. It applies to new components, component refactors, public API design, and tests across all modules in this repository.

The goal is to keep components consistent, maintainable, and predictable for both users and maintainers.

## Design Principles

- Keep component boundaries clear. A component should own its configuration, runtime behavior, and tests within its package or module boundary.
- Prefer explicit behavior over implicit side effects. Errors should be returned where they occur and should not be hidden until later runtime behavior.
- Keep public APIs small and stable. Add exported identifiers only when they describe a real user-facing capability.
- Prefer established repository patterns over new local styles unless the component has a clear reason to differ.
- Avoid over-design. Add abstractions only when they reduce real complexity, remove meaningful duplication, or make public behavior easier to understand.

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
