package health

import (
	"fmt"
	"time"
)

// PanicError describes a panic recovered while running a [Checker].
type PanicError struct {
	Value any
	Stack []byte
}

// Error returns a description of the recovered panic.
func (e *PanicError) Error() string {
	return fmt.Sprintf("health: checker panic: %v", e.Value)
}

// Unwrap returns the recovered value when it is an error.
func (e *PanicError) Unwrap() error {
	err, _ := e.Value.(error)
	return err
}

// Result describes one named health check.
type Result struct {
	Name     string
	Duration time.Duration
	Err      error
}

// Healthy reports whether the check completed without an error.
func (r Result) Healthy() bool {
	return r.Err == nil
}

// Report describes one complete [Registry.Check].
type Report struct {
	StartedAt time.Time
	Duration  time.Duration
	Results   []Result
}

// Healthy reports whether every result is healthy. An empty report is healthy.
func (r Report) Healthy() bool {
	for _, result := range r.Results {
		if !result.Healthy() {
			return false
		}
	}
	return true
}
