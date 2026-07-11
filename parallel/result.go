package parallel

// Result contains the outcome of processing one input value.
type Result[T any] struct {
	// Value is the callback result. It is the zero value of T when the callback
	// fails before producing a value.
	Value T
	// Err is the callback error for the corresponding input value.
	Err error
}
