package signal

import (
	"context"
	"os"
)

// Handler processes an operating system signal. Returning an error reports a
// failed operation; it does not stop the Router or retry the operation.
type Handler interface {
	Handle(context.Context, os.Signal) error
}

// HandlerFunc adapts a function to [Handler].
type HandlerFunc func(context.Context, os.Signal) error

// Handle calls f with the handler context and received signal.
func (f HandlerFunc) Handle(ctx context.Context, sig os.Signal) error {
	return f(ctx, sig)
}
