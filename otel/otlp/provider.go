package otlp

import (
	"context"

	"github.com/go-fries/fries/foundation/v3"
)

// Provider adapts a [Client] to the [foundation.Provider] lifecycle.
type Provider struct {
	client *Client
}

var _ foundation.Provider = (*Provider)(nil)

// NewProvider creates a [Provider] backed by client.
func NewProvider(client *Client) *Provider {
	return &Provider{
		client: client,
	}
}

// Bootstrap configures the underlying client.
func (p *Provider) Bootstrap(ctx context.Context) (context.Context, error) {
	return ctx, p.client.Configure(ctx)
}

// Terminate shuts down the underlying client.
func (p *Provider) Terminate(ctx context.Context) (context.Context, error) {
	return ctx, p.client.Shutdown(ctx)
}
