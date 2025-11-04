package provider

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProvider is a mock implementation of Provider for testing
type mockProvider struct {
	bootstrapFunc func(context.Context) (context.Context, error)
	terminateFunc func(context.Context) (context.Context, error)
	name          string
}

func (m *mockProvider) Bootstrap(ctx context.Context) (context.Context, error) {
	if m.bootstrapFunc != nil {
		return m.bootstrapFunc(ctx)
	}
	return ctx, nil
}

func (m *mockProvider) Terminate(ctx context.Context) (context.Context, error) {
	if m.terminateFunc != nil {
		return m.terminateFunc(ctx)
	}
	return ctx, nil
}

// contextKey is used for storing values in context during tests
type contextKey string

func TestNewProviders(t *testing.T) {
	t.Run("empty providers", func(t *testing.T) {
		providers := NewProviders()
		assert.Empty(t, providers)
	})

	t.Run("single provider", func(t *testing.T) {
		p := &mockProvider{name: "test"}
		providers := NewProviders(p)
		assert.Len(t, providers, 1)
	})

	t.Run("multiple providers", func(t *testing.T) {
		p1 := &mockProvider{name: "test1"}
		p2 := &mockProvider{name: "test2"}
		p3 := &mockProvider{name: "test3"}
		providers := NewProviders(p1, p2, p3)
		assert.Len(t, providers, 3)
	})
}

func TestProviders_Bootstrap(t *testing.T) {
	t.Run("empty providers", func(t *testing.T) {
		providers := NewProviders()
		ctx := context.Background()
		resultCtx, err := providers.Bootstrap(ctx)
		assert.NoError(t, err)
		assert.Equal(t, ctx, resultCtx)
	})

	t.Run("single provider success", func(t *testing.T) {
		called := false
		p := &mockProvider{
			bootstrapFunc: func(ctx context.Context) (context.Context, error) {
				called = true
				return context.WithValue(ctx, contextKey("key1"), "value1"), nil
			},
		}
		providers := NewProviders(p)
		ctx := context.Background()
		resultCtx, err := providers.Bootstrap(ctx)
		assert.NoError(t, err)
		assert.True(t, called, "expected provider to be called")
		assert.Equal(t, "value1", resultCtx.Value(contextKey("key1")))
	})

	t.Run("multiple providers success with order", func(t *testing.T) {
		var order []string
		p1 := &mockProvider{
			bootstrapFunc: func(ctx context.Context) (context.Context, error) {
				order = append(order, "p1")
				return context.WithValue(ctx, contextKey("p1"), "v1"), nil
			},
		}
		p2 := &mockProvider{
			bootstrapFunc: func(ctx context.Context) (context.Context, error) {
				order = append(order, "p2")
				return context.WithValue(ctx, contextKey("p2"), "v2"), nil
			},
		}
		p3 := &mockProvider{
			bootstrapFunc: func(ctx context.Context) (context.Context, error) {
				order = append(order, "p3")
				return context.WithValue(ctx, contextKey("p3"), "v3"), nil
			},
		}

		providers := NewProviders(p1, p2, p3)
		ctx := context.Background()
		resultCtx, err := providers.Bootstrap(ctx)
		assert.NoError(t, err)

		// Verify execution order
		assert.Equal(t, []string{"p1", "p2", "p3"}, order)

		// Verify all context values are set
		assert.Equal(t, "v1", resultCtx.Value(contextKey("p1")))
		assert.Equal(t, "v2", resultCtx.Value(contextKey("p2")))
		assert.Equal(t, "v3", resultCtx.Value(contextKey("p3")))
	})

	t.Run("provider returns error", func(t *testing.T) {
		expectedErr := errors.New("bootstrap failed")
		p1 := &mockProvider{
			bootstrapFunc: func(ctx context.Context) (context.Context, error) {
				return ctx, nil
			},
		}
		p2 := &mockProvider{
			bootstrapFunc: func(ctx context.Context) (context.Context, error) {
				return ctx, expectedErr
			},
		}
		p3 := &mockProvider{
			bootstrapFunc: func(ctx context.Context) (context.Context, error) {
				t.Error("p3 should not be called")
				return ctx, nil
			},
		}

		providers := NewProviders(p1, p2, p3)
		ctx := context.Background()
		_, err := providers.Bootstrap(ctx)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestProviders_Terminate(t *testing.T) {
	t.Run("empty providers", func(t *testing.T) {
		providers := NewProviders()
		ctx := context.Background()
		resultCtx, err := providers.Terminate(ctx)
		assert.NoError(t, err)
		assert.Equal(t, ctx, resultCtx)
	})

	t.Run("single provider success", func(t *testing.T) {
		called := false
		p := &mockProvider{
			terminateFunc: func(ctx context.Context) (context.Context, error) {
				called = true
				return context.WithValue(ctx, contextKey("terminated"), true), nil
			},
		}
		providers := NewProviders(p)
		ctx := context.Background()
		resultCtx, err := providers.Terminate(ctx)
		assert.NoError(t, err)
		assert.True(t, called, "expected provider to be called")
		assert.Equal(t, true, resultCtx.Value(contextKey("terminated")))
	})

	t.Run("multiple providers success with reverse order", func(t *testing.T) {
		var order []string
		p1 := &mockProvider{
			terminateFunc: func(ctx context.Context) (context.Context, error) {
				order = append(order, "p1")
				return ctx, nil
			},
		}
		p2 := &mockProvider{
			terminateFunc: func(ctx context.Context) (context.Context, error) {
				order = append(order, "p2")
				return ctx, nil
			},
		}
		p3 := &mockProvider{
			terminateFunc: func(ctx context.Context) (context.Context, error) {
				order = append(order, "p3")
				return ctx, nil
			},
		}

		providers := NewProviders(p1, p2, p3)
		ctx := context.Background()
		_, err := providers.Terminate(ctx)
		assert.NoError(t, err)

		// Verify execution order (reverse)
		assert.Equal(t, []string{"p3", "p2", "p1"}, order)
	})

	t.Run("provider returns error", func(t *testing.T) {
		expectedErr := errors.New("terminate failed")
		p1 := &mockProvider{
			terminateFunc: func(ctx context.Context) (context.Context, error) {
				t.Error("p1 should not be called")
				return ctx, nil
			},
		}
		p2 := &mockProvider{
			terminateFunc: func(ctx context.Context) (context.Context, error) {
				return ctx, expectedErr
			},
		}
		p3 := &mockProvider{
			terminateFunc: func(ctx context.Context) (context.Context, error) {
				return ctx, nil
			},
		}

		providers := NewProviders(p1, p2, p3)
		ctx := context.Background()
		_, err := providers.Terminate(ctx)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestProviders_Build(t *testing.T) {
	t.Run("success with cleanup", func(t *testing.T) {
		var bootstrapCalled, terminateCalled bool
		p := &mockProvider{
			bootstrapFunc: func(ctx context.Context) (context.Context, error) {
				bootstrapCalled = true
				return context.WithValue(ctx, contextKey("built"), true), nil
			},
			terminateFunc: func(ctx context.Context) (context.Context, error) {
				terminateCalled = true
				return ctx, nil
			},
		}

		providers := NewProviders(p)
		ctx := context.Background()
		resultCtx, cleanup, err := providers.Build(ctx)
		require.NoError(t, err)
		require.NotNil(t, cleanup)
		assert.True(t, bootstrapCalled, "expected bootstrap to be called")
		assert.Equal(t, true, resultCtx.Value(contextKey("built")))

		// Call cleanup
		_, cleanupErr := cleanup()
		assert.NoError(t, cleanupErr)
		assert.True(t, terminateCalled, "expected terminate to be called")
	})

	t.Run("bootstrap fails", func(t *testing.T) {
		expectedErr := errors.New("bootstrap failed")
		p := &mockProvider{
			bootstrapFunc: func(ctx context.Context) (context.Context, error) {
				return ctx, expectedErr
			},
		}

		providers := NewProviders(p)
		ctx := context.Background()
		_, cleanup, err := providers.Build(ctx)
		assert.ErrorIs(t, err, expectedErr)
		assert.Nil(t, cleanup)
	})

	t.Run("cleanup fails", func(t *testing.T) {
		expectedErr := errors.New("terminate failed")
		p := &mockProvider{
			bootstrapFunc: func(ctx context.Context) (context.Context, error) {
				return ctx, nil
			},
			terminateFunc: func(ctx context.Context) (context.Context, error) {
				return ctx, expectedErr
			},
		}

		providers := NewProviders(p)
		ctx := context.Background()
		_, cleanup, err := providers.Build(ctx)
		require.NoError(t, err)
		require.NotNil(t, cleanup)

		// Call cleanup and expect error
		_, cleanupErr := cleanup()
		assert.ErrorIs(t, cleanupErr, expectedErr)
	})

	t.Run("multiple providers with cleanup", func(t *testing.T) {
		var bootstrapOrder, terminateOrder []string
		p1 := &mockProvider{
			bootstrapFunc: func(ctx context.Context) (context.Context, error) {
				bootstrapOrder = append(bootstrapOrder, "p1")
				return ctx, nil
			},
			terminateFunc: func(ctx context.Context) (context.Context, error) {
				terminateOrder = append(terminateOrder, "p1")
				return ctx, nil
			},
		}
		p2 := &mockProvider{
			bootstrapFunc: func(ctx context.Context) (context.Context, error) {
				bootstrapOrder = append(bootstrapOrder, "p2")
				return ctx, nil
			},
			terminateFunc: func(ctx context.Context) (context.Context, error) {
				terminateOrder = append(terminateOrder, "p2")
				return ctx, nil
			},
		}

		providers := NewProviders(p1, p2)
		ctx := context.Background()
		_, cleanup, err := providers.Build(ctx)
		assert.NoError(t, err)

		// Verify bootstrap order
		assert.Equal(t, []string{"p1", "p2"}, bootstrapOrder)

		// Call cleanup
		_, err = cleanup()
		assert.NoError(t, err)

		// Verify terminate order (reverse)
		assert.Equal(t, []string{"p2", "p1"}, terminateOrder)
	})
}
