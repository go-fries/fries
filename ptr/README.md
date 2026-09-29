# Ptr

`ptr` reads optional pointer values. It is useful when an API distinguishes an omitted value from a value's zero value. With Go 1.26, use `new(value)` to construct pointers directly; `ptr.Ptr` remains available for compatibility but is deprecated.

## Installation

```bash
go get github.com/go-fries/fries/ptr/v4
```

## Usage

```go
package main

type Config struct {
	Name    *string
	Enabled *bool
}

func config() Config {
	return Config{
		Name:    new("fries"),
		Enabled: new(true),
	}
}
```

`new(value)` returns a pointer to a new variable initialized with the supplied value, including zero values and typed nil pointers.

## Read a pointer

`Value` returns the pointed-to value and reports whether the pointer is non-nil:

```go
name, ok := ptr.Value(config.Name)
```

For a nil pointer, `Value` returns the type's zero value and `false`. A non-nil pointer to a zero value returns that zero value and `true`.

## Apply a fallback

`Or` returns the pointed-to value, or the supplied fallback when the pointer is nil:

```go
name := ptr.Or(config.Name, "fries")
```

The fallback is used only when the pointer itself is nil. A non-nil pointer to a zero value does not use the fallback.
