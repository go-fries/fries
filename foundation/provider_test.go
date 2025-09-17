package foundation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type providerFuncKey struct{}

func TestProvider_BootstrapFunc(t *testing.T) {
	p := BootstrapFunc(func(ctx context.Context) (context.Context, error) {
		return context.WithValue(ctx, providerFuncKey{}, "test"), nil
	})

	ctx := t.Context()
	ctx, err := p.Bootstrap(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "test", ctx.Value(providerFuncKey{}))

	ctx, err = p.Terminate(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "test", ctx.Value(providerFuncKey{}))
}

func TestProvider_BootstrapFuncError(t *testing.T) {
	p := TerminateFunc(func(ctx context.Context) (context.Context, error) {
		return context.WithValue(ctx, providerFuncKey{}, "test"), nil
	})

	ctx := t.Context()
	ctx, err := p.Bootstrap(ctx)
	assert.NoError(t, err)
	assert.Nil(t, ctx.Value(providerFuncKey{}))

	ctx, err = p.Terminate(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "test", ctx.Value(providerFuncKey{}))
}

func TestProvider_Chain(t *testing.T) {
	p1 := BootstrapFunc(func(ctx context.Context) (context.Context, error) {
		return context.WithValue(ctx, providerFuncKey{}, "test1"), nil
	})
	p2 := BootstrapFunc(func(ctx context.Context) (context.Context, error) {
		return context.WithValue(ctx, providerFuncKey{}, "test2"), nil
	})
	p3 := TerminateFunc(func(ctx context.Context) (context.Context, error) {
		return context.WithValue(ctx, providerFuncKey{}, "test3"), nil
	})

	c := NewChain(p1, p2, p3)
	ctx := t.Context()

	ctx, err := c.Bootstrap(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "test2", ctx.Value(providerFuncKey{}))

	ctx, err = c.Terminate(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "test3", ctx.Value(providerFuncKey{}))
}
