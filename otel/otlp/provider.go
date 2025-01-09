package otlp

import (
	"context"

	foundation "github.com/go-fries/fries/foundation/v3"
)

type Provider struct {
	client *Client
}

var _ foundation.Provider = (*Provider)(nil)

func NewProvider(client *Client) *Provider {
	return &Provider{
		client: client,
	}
}

func (p *Provider) Bootstrap(ctx context.Context) (context.Context, error) {
	return ctx, p.client.Configure(ctx)
}

func (p *Provider) Terminate(ctx context.Context) (context.Context, error) {
	return ctx, p.client.Shutdown(ctx)
}
