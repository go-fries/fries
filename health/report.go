package health

import "time"

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
