package otlp

import (
	"context"

	"github.com/go-fries/fries/lifecycle/v4"
)

// Provider adapts a [Client] to the [lifecycle.Provider] lifecycle.
type Provider struct {
	client *Client
}

var _ lifecycle.Provider = (*Provider)(nil)

// NewProvider creates a [Provider] backed by client.
func NewProvider(client *Client) *Provider {
	return &Provider{
		client: client,
	}
}

// Bootstrap configures the underlying client.
func (p *Provider) Bootstrap(ctx context.Context) (context.Context, error) {
	if p.client == nil {
		return ctx, ErrClientRequired
	}
	return ctx, p.client.Configure(ctx)
}

// Shutdown shuts down the underlying client.
func (p *Provider) Shutdown(ctx context.Context) (context.Context, error) {
	if p.client == nil {
		return ctx, ErrClientRequired
	}
	return ctx, p.client.Shutdown(ctx)
}
