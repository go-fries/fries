package event

import "context"

type contextKey struct{}

// WithDispatcher returns a copy of ctx that carries dispatcher.
// WithDispatcher panics if ctx or dispatcher is nil.
func WithDispatcher(ctx context.Context, dispatcher *Dispatcher) context.Context {
	if ctx == nil {
		panic("event: nil context")
	}
	if dispatcher == nil {
		panic("event: nil dispatcher")
	}
	return context.WithValue(ctx, contextKey{}, dispatcher)
}

// FromContext returns the Dispatcher carried by ctx, if any.
// FromContext panics if ctx is nil.
func FromContext(ctx context.Context) (*Dispatcher, bool) {
	if ctx == nil {
		panic("event: nil context")
	}
	dispatcher, ok := ctx.Value(contextKey{}).(*Dispatcher)
	return dispatcher, ok
}
