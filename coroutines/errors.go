package coroutines

import "errors"

var (
	// ErrInvalidLimit indicates that a concurrency limit is not positive.
	ErrInvalidLimit = errors.New("coroutines: concurrency limit must be greater than zero")
	// ErrNilTask indicates that a task passed to Run or RunLimit is nil.
	ErrNilTask = errors.New("coroutines: task must not be nil")
	// ErrNilFunc indicates that a callback passed to ForEach or Map is nil.
	ErrNilFunc = errors.New("coroutines: function must not be nil")
)
