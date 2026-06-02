# Contributing

## Component Configuration Design
For new components and refactors, prefer keeping configuration state and option parsing in a package-local `config.go`. The file should define the internal `config` type, default values, the public `Option` contract, `optionFunc`, `WithXxx` option helpers, `newConfig(...)`, and small config-owned factory helpers such as `newLogger(...)`, `newResource(...)`, or `newTracerProvider(...)` when they are driven by configuration.

Keep runtime types focused on lifecycle and behavior. For example, a `Client`, `Logger`, or `Middleware` should generally call into its `config` for configured dependencies instead of carrying many option fields or scattered defaulting logic directly.

Use an interface-based option contract with an unexported apply method:

```go
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}
```

Then implement public options as `WithXxx(...) Option` functions that return `optionFunc`. Place these option functions immediately after `optionFunc.apply`, followed by `newConfig(...)` and config helper methods. Prefer this form over exporting or aliasing a raw `func(*config)` option type because it keeps option construction controlled inside the package while still allowing future specialized option implementations.

When a component needs option-time validation or setup that can fail, it is acceptable for that package to use an error-returning option contract instead:

```go
type Option interface {
	apply(*config) error
}
```

Choose the non-error form for simple assignment and defaulting. Choose the error-returning form only when the component has a real failure path during option application, and keep validation errors explicit rather than hiding them in later runtime behavior.
