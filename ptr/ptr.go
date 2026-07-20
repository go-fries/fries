package ptr

// Ptr returns a pointer to value.
func Ptr[T any](value T) *T {
	return &value
}

// Value returns the value pointed to by value and reports whether value is non-nil.
func Value[T any](value *T) (T, bool) {
	if value != nil {
		return *value, true
	}

	var zero T
	return zero, false
}

// Or returns the value pointed to by value, or fallback when value is nil.
func Or[T any](value *T, fallback T) T {
	if value != nil {
		return *value
	}

	return fallback
}
