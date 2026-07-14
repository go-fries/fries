package env

import (
	"context"
)

type Provider struct {
	env Env
}

func NewProvider(env Env) *Provider {
	return &Provider{env: env}
}

func (p *Provider) Bootstrap(ctx context.Context) (context.Context, error) {
	SetEnv(p.env)
	return NewContext(ctx, p.env), nil
}

func (p *Provider) Shutdown(ctx context.Context) (context.Context, error) {
	return ctx, nil
}
