package parallel

import "errors"

var (
	// ErrInvalidLimit indicates that a concurrency limit is not positive.
	ErrInvalidLimit = errors.New("parallel: concurrency limit must be greater than zero")
	// ErrNilTask indicates that a task passed to Run, RunLimit, Submit, or
	// Execute is nil.
	ErrNilTask = errors.New("parallel: task must not be nil")
	// ErrNilFunc indicates that a callback passed to ForEach, Map, MapResults,
	// or Filter is nil.
	ErrNilFunc = errors.New("parallel: function must not be nil")
	// ErrPoolClosed indicates that a pool no longer accepts tasks.
	ErrPoolClosed = errors.New("parallel: pool is closed")
)
