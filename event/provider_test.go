package event

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type lifecycleProvider interface {
	Bootstrap(context.Context) (context.Context, error)
	Shutdown(context.Context) (context.Context, error)
}

var _ lifecycleProvider = (*Provider)(nil)

func TestProvider(t *testing.T) {
	dispatcher := New()
	provider := NewProvider(dispatcher)

	ctx, err := provider.Bootstrap(t.Context())
	require.NoError(t, err)
	actual, ok := FromContext(ctx)
	require.True(t, ok)
	assert.Same(t, dispatcher, actual)

	shutdownCtx, err := provider.Shutdown(ctx)
	require.NoError(t, err)
	assert.Same(t, ctx, shutdownCtx)
}

func TestNewProviderPanicsForNilDispatcher(t *testing.T) {
	assert.Panics(t, func() { NewProvider(nil) })
}
