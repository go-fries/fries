package foundation

import (
	"context"
	"slices"
)

type Provider interface {
	Bootstrap(context.Context) (context.Context, error)
	Terminate(context.Context) (context.Context, error)
}

// Chain is a provider that calls multiple providers in sequence.
type Chain []Provider

func NewChain(providers ...Provider) Chain {
	return providers
}

func (c Chain) Bootstrap(ctx context.Context) (context.Context, error) {
	var err error
	for _, p := range c {
		ctx, err = p.Bootstrap(ctx)
		if err != nil {
			return ctx, err
		}
	}

	return ctx, nil
}

func (c Chain) Terminate(ctx context.Context) (context.Context, error) {
	var err error
	for _, provider := range slices.Backward(c) {
		ctx, err = provider.Terminate(ctx)
		if err != nil {
			return ctx, err
		}
	}

	return ctx, nil
}

// BaseProvider is a provider that does nothing.
type BaseProvider struct{}

func (*BaseProvider) Bootstrap(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

func (*BaseProvider) Terminate(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

type BootstrapFunc func(context.Context) (context.Context, error)

func (f BootstrapFunc) Bootstrap(ctx context.Context) (context.Context, error) {
	return f(ctx)
}

func (f BootstrapFunc) Terminate(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

type TerminateFunc func(context.Context) (context.Context, error)

func (f TerminateFunc) Bootstrap(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

func (f TerminateFunc) Terminate(ctx context.Context) (context.Context, error) {
	return f(ctx)
}
