package support

// If returns trueVal if condition is true, otherwise falseVal.
//
//	If(true, "foo", "bar") // "foo"
//	If(false, "foo", "bar") // "bar"
//
// Warning: that the trueVal and falseVal in this method will
// also be executed during runtime, and there may be panics
// during use. Please be cautious when using it.
// You can use IfFunc or Optional to avoid this situation.
//
// Example:
//
//	var nilVal *foo
//	If(nilVal != nil, nilVal.Name, "") // panic: runtime error: invalid memory address or nil pointer dereference
//	If(nilVal != nil, Optional(nilVal).Name, "") // ""
func If[T any](condition bool, trueVal T, falseVal T) T {
	if condition {
		return trueVal
	}

	return falseVal
}

// IfFunc returns trueFunc() if condition is true, otherwise falseFunc().
//
//	IfFunc(true, func() string {
//		return "foo"
//	}, func() string {
//		return "bar"
//	}) // "foo"
func IfFunc[T any](condition bool, trueFunc func() T, falseFunc func() T) T {
	if condition {
		return trueFunc()
	}

	return falseFunc()
}

// Unless returns falseVal if condition is true, otherwise trueVal.
//
//	Unless(true, "foo", "bar") // "bar"
//	Unless(false, "foo", "bar") // "foo"
func Unless[T any](condition bool, falseVal T, trueVal T) T {
	return If(condition, trueVal, falseVal)
}

// Optional returns the value if it is not nil, otherwise the zero value.
//
//	Optional(&foo{Name: "bar"}) // &foo{Name: "bar"}
//	Optional[foo](nil) // &foo{}
//	Optional[int](nil) // *int(0)
func Optional[T any](value *T) *T {
	if value != nil {
		return value
	}
	var zero T
	return &zero
}

// Tap calls the given callback with the given value then returns the value.
//
//	Tap("foo", func(s string) {
//		fmt.Println(s) // "foo" and os.Stdout will print "foo"
//	}, func(s string) {
//		// more callbacks
//	}...)
func Tap[T any](value T, callbacks ...func(T)) T {
	for _, callback := range callbacks {
		if callback != nil {
			callback(value)
		}
	}

	return value
}

// With calls the given callbacks with the given value then return the value.
//
//	With("foo", func(s string) string {
//		return s + "bar"
//	}, func(s string) string {
//		return s + "baz"
//	}) // "foobarbaz"
func With[T any](value T, callbacks ...func(T) T) T {
	for _, callback := range callbacks {
		if callback != nil {
			value = callback(value)
		}
	}

	return value
}

// Transform calls the given callback with the given value then return the result.
//
//	Transform(1, strconv.Itoa) // "1"
//	Transform("foo", func(s string) *foo {
//		return &foo{Name: s}
//	}) // &foo{Name: "foo"}
func Transform[T, R any](value T, callback func(T) R) R {
	return callback(value)
}

// ReturnIf returns the result of the callback if the condition is true.
// Otherwise, return the default value or zero value.
//
//	ReturnIf(true, func() string {
//		return "foo"
//	}, "bar") // "foo"
//	ReturnIf(false, func() string {
//		return "foo"
//	}, "bar") // "bar"
func ReturnIf[T any](condition bool, callback func() T, defaults ...T) T {
	if condition {
		return callback()
	}

	if len(defaults) > 0 {
		return defaults[0]
	}

	var zero T
	return zero
}

// Default returns the first non-zero value.
// If all values are zero, return the zero value.
//
//	Default("", "foo") // "foo"
//	Default("bar", "foo") // "bar"
//	Default("", "", "foo") // "foo"
func Default[T comparable](values ...T) T {
	var zero T
	for _, value := range values {
		if value != zero {
			return value
		}
	}
	return zero
}

func DefaultFunc[T comparable](callbacks ...func() T) T {
	var zero, value T
	for _, callback := range callbacks {
		if callback != nil {
			value = callback()
			if value != zero {
				return value
			}
		}
	}
	return zero
}

// DefaultWithFunc returns defaultValue if value is zero, otherwise value.
//
//	DefaultWithFunc("", func() string { return "foo" }) // "foo"
//	DefaultWithFunc("bar", func() string { return "foo" }) // "bar"
//	DefaultWithFunc("", func() string { return "" }, func() string { return "foo" }) // "foo"
func DefaultWithFunc[T comparable](value T, callbacks ...func() T) T {
	var zero T
	if value != zero {
		return value
	}
	for _, callback := range callbacks {
		if callback != nil {
			value = callback()
			if value != zero {
				return value
			}
		}
	}
	return zero
}

// Ptr returns a pointer to the value.
//
//	Ptr("foo") // *string("foo")
//	Ptr(1) // *int(1)
func Ptr[T any](value T) *T {
	return &value
}

// Val returns the value of the pointer.
// If the pointer is nil, return the zero value.
//
//	Val((*string)(nil)) // ""
//	Val(Ptr("foo")) // "foo"
func Val[T any](value *T) T {
	if value != nil {
		return *value
	}
	var zero T
	return zero
}

// IsType returns true if the value is the type.
//
//	IsType[int](1) // true
//	IsType[int]("foo") // false
func IsType[T any](value any) bool {
	_, ok := value.(T)
	return ok
}

// IsZero returns true if the value is zero.
//
//	IsZero(0) // true
//	IsZero("") // true
//	IsZero("foo") // false
func IsZero[T comparable](value T) bool {
	var zero T
	return value == zero
}

// IsEmpty returns true if the value is zero.
//
//	IsEmpty(0) // true
//	IsEmpty("") // true
func IsEmpty[T comparable](value T) bool {
	return IsZero(value)
}
