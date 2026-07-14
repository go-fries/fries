package lifecycle

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ Provider = (*Runner)(nil)

func TestBootstrapFunc(t *testing.T) {
	provider := BootstrapFunc(func(ctx context.Context) (context.Context, error) {
		return context.WithValue(ctx, contextKey("bootstrap"), true), nil
	})

	ctx, err := provider.Bootstrap(t.Context())
	require.NoError(t, err)
	assert.Equal(t, true, ctx.Value(contextKey("bootstrap")))

	next, err := provider.Shutdown(ctx)
	require.NoError(t, err)
	assert.Same(t, ctx, next)
}

func TestShutdownFunc(t *testing.T) {
	provider := ShutdownFunc(func(ctx context.Context) (context.Context, error) {
		return context.WithValue(ctx, contextKey("shutdown"), true), nil
	})

	ctx, err := provider.Bootstrap(t.Context())
	require.NoError(t, err)
	assert.Same(t, t.Context(), ctx)

	next, err := provider.Shutdown(ctx)
	require.NoError(t, err)
	assert.Equal(t, true, next.Value(contextKey("shutdown")))
}
