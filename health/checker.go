package health

import "context"

// Checker reports whether one application dependency or capability is
// healthy. A nil error reports a healthy result.
type Checker interface {
	Check(context.Context) error
}

// CheckFunc adapts a function to [Checker].
type CheckFunc func(context.Context) error

// Check calls f with ctx.
func (f CheckFunc) Check(ctx context.Context) error {
	return f(ctx)
}
