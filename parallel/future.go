package parallel

import "context"

// Future represents the eventual result of a task accepted by a Pool.
//
// A Future may be waited on multiple times. Canceling a Wait context stops
// only that wait; it does not cancel the task. Future values are created by a
// Pool; the zero value is not valid.
type Future struct {
	done chan struct{}
	err  error
}

func newFuture() *Future {
	return &Future{done: make(chan struct{})}
}

// Done returns a channel that is closed when the task finishes.
func (f *Future) Done() <-chan struct{} {
	return f.done
}

// Wait waits for the task to finish or ctx to be canceled.
func (f *Future) Wait(ctx context.Context) error {
	select {
	case <-f.done:
		return f.err
	default:
	}

	select {
	case <-f.done:
		return f.err
	case <-ctx.Done():
		select {
		case <-f.done:
			return f.err
		default:
			return context.Cause(ctx)
		}
	}
}

func (f *Future) complete(err error) {
	f.err = err
	close(f.done)
}
