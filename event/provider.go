package event

import "context"

// Provider adds a Dispatcher to an application lifecycle context.
type Provider struct {
	dispatcher *Dispatcher
}

// NewProvider creates a Provider for dispatcher. It panics if dispatcher is
// nil.
func NewProvider(dispatcher *Dispatcher) *Provider {
	if dispatcher == nil {
		panic("event: nil dispatcher")
	}
	return &Provider{dispatcher: dispatcher}
}

// Bootstrap sets the configured Dispatcher as the default and adds it to ctx.
func (p *Provider) Bootstrap(ctx context.Context) (context.Context, error) {
	SetDefault(p.dispatcher)
	return NewContext(ctx, p.dispatcher), nil
}

// Shutdown returns ctx unchanged. Dispatcher operations are synchronous, so no
// background event work needs to be drained.
func (p *Provider) Shutdown(ctx context.Context) (context.Context, error) {
	return ctx, nil
}
