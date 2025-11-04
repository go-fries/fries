package provider

import "context"

type Provider interface {
	// Bootstrap initializes the provider and returns a new context.
	Bootstrap(context.Context) (context.Context, error)

	// Terminate cleans up the provider and returns a new context.
	Terminate(context.Context) (context.Context, error)
}

type Providers []Provider

func NewProviders(providers ...Provider) Providers {
	return providers
}

func (p Providers) Bootstrap(ctx context.Context) (context.Context, error) {
	var err error
	for _, provider := range p {
		ctx, err = provider.Bootstrap(ctx)
		if err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}

func (p Providers) Terminate(ctx context.Context) (context.Context, error) {
	var err error
	for i := len(p) - 1; i >= 0; i-- {
		ctx, err = p[i].Terminate(ctx)
		if err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}

func (p Providers) Build(ctx context.Context) (context.Context, func() (context.Context, error), error) {
	var err error
	ctx, err = p.Bootstrap(ctx)
	if err != nil {
		return ctx, nil, err
	}

	cleanup := func() (context.Context, error) {
		return p.Terminate(ctx)
	}

	return ctx, cleanup, nil
}
