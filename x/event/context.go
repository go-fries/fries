package event

import "context"

type contextKey struct{}

func NewContext(ctx context.Context, d *Dispatcher) context.Context {
	return context.WithValue(ctx, contextKey{}, d)
}

func FromContext(ctx context.Context) (*Dispatcher, bool) {
	d, ok := ctx.Value(contextKey{}).(*Dispatcher)
	return d, ok
}
