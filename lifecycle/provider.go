package lifecycle

import "context"

// Provider participates in application startup and shutdown.
//
// Bootstrap receives the context returned by the preceding provider and may
// return a derived context for the next provider. Shutdown runs in reverse
// order with the same propagation rule. A successful call must return a
// non-nil context derived from the context it received.
type Provider interface {
	Bootstrap(context.Context) (context.Context, error)
	Shutdown(context.Context) (context.Context, error)
}

// BootstrapFunc adapts a bootstrap function into a [Provider]. Its Shutdown
// method returns the supplied context unchanged.
type BootstrapFunc func(context.Context) (context.Context, error)

// Bootstrap calls f with ctx.
func (f BootstrapFunc) Bootstrap(ctx context.Context) (context.Context, error) {
	return f(ctx)
}

// Shutdown returns ctx unchanged.
func (BootstrapFunc) Shutdown(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

// ShutdownFunc adapts a shutdown function into a [Provider]. Its Bootstrap
// method returns the supplied context unchanged.
type ShutdownFunc func(context.Context) (context.Context, error)

// Bootstrap returns ctx unchanged.
func (ShutdownFunc) Bootstrap(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

// Shutdown calls f with ctx.
func (f ShutdownFunc) Shutdown(ctx context.Context) (context.Context, error) {
	return f(ctx)
}
