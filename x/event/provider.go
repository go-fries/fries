package event

import (
	"context"
)

type Provider struct {
	*Dispatcher
}

func NewProvider(dispatcher *Dispatcher) *Provider {
	return &Provider{
		Dispatcher: dispatcher,
	}
}

func (p *Provider) Bootstrap(ctx context.Context) (context.Context, error) {
	return NewContext(ctx, p.Dispatcher), nil
}

func (p *Provider) Terminate(ctx context.Context) (context.Context, error) {
	p.Dispatcher.Wait() // wait for all events to be processed
	return ctx, nil
}
