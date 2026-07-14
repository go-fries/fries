package lifecycle

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testProvider struct {
	bootstrap func(context.Context) (context.Context, error)
	shutdown  func(context.Context) (context.Context, error)
}

func (p *testProvider) Bootstrap(ctx context.Context) (context.Context, error) {
	if p.bootstrap == nil {
		return ctx, nil
	}
	return p.bootstrap(ctx)
}

func (p *testProvider) Shutdown(ctx context.Context) (context.Context, error) {
	if p.shutdown == nil {
		return ctx, nil
	}
	return p.shutdown(ctx)
}

type contextKey string

func TestRunnerRun(t *testing.T) {
	var order []string
	providerA := &testProvider{
		bootstrap: func(ctx context.Context) (context.Context, error) {
			order = append(order, "bootstrap-a")
			return context.WithValue(ctx, contextKey("a"), "a"), nil
		},
		shutdown: func(ctx context.Context) (context.Context, error) {
			order = append(order, "shutdown-a")
			assert.Equal(t, "a", ctx.Value(contextKey("a")))
			assert.Equal(t, "b", ctx.Value(contextKey("b")))
			assert.Equal(t, "shutdown-b", ctx.Value(contextKey("shutdown")))
			return ctx, nil
		},
	}
	providerB := &testProvider{
		bootstrap: func(ctx context.Context) (context.Context, error) {
			order = append(order, "bootstrap-b")
			assert.Equal(t, "a", ctx.Value(contextKey("a")))
			return context.WithValue(ctx, contextKey("b"), "b"), nil
		},
		shutdown: func(ctx context.Context) (context.Context, error) {
			order = append(order, "shutdown-b")
			assert.Equal(t, "a", ctx.Value(contextKey("a")))
			assert.Equal(t, "b", ctx.Value(contextKey("b")))
			return context.WithValue(ctx, contextKey("shutdown"), "shutdown-b"), nil
		},
	}

	runner := New(WithProviders(providerA, providerB))
	err := runner.Run(t.Context(), func(ctx context.Context) error {
		order = append(order, "handler")
		assert.Equal(t, "a", ctx.Value(contextKey("a")))
		assert.Equal(t, "b", ctx.Value(contextKey("b")))
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, []string{
		"bootstrap-a",
		"bootstrap-b",
		"handler",
		"shutdown-b",
		"shutdown-a",
	}, order)
}

func TestRunnerBootstrapAndShutdown(t *testing.T) {
	var order []string
	provider := &testProvider{
		bootstrap: func(ctx context.Context) (context.Context, error) {
			order = append(order, "bootstrap")
			return context.WithValue(ctx, contextKey("runtime"), true), nil
		},
		shutdown: func(ctx context.Context) (context.Context, error) {
			order = append(order, "shutdown")
			assert.ErrorIs(t, ctx.Err(), context.Canceled)
			assert.Equal(t, true, ctx.Value(contextKey("runtime")))
			return context.WithValue(ctx, contextKey("shutdown"), true), nil
		},
	}

	runner := New(WithProviders(provider))
	bootstrap := runner.Bootstrap
	shutdown := runner.Shutdown

	runtimeCtx, err := bootstrap(t.Context())
	require.NoError(t, err)
	assert.Equal(t, true, runtimeCtx.Value(contextKey("runtime")))

	shutdownInput, cancel := context.WithCancel(runtimeCtx)
	cancel()
	shutdownCtx, err := shutdown(shutdownInput)
	require.NoError(t, err)
	assert.Equal(t, true, shutdownCtx.Value(contextKey("shutdown")))
	assert.Equal(t, []string{"bootstrap", "shutdown"}, order)
}

func TestRunnerShutdownLifecycle(t *testing.T) {
	t.Run("before bootstrap", func(t *testing.T) {
		runner := New()

		ctx, err := runner.Shutdown(t.Context())
		require.NoError(t, err)
		assert.Same(t, t.Context(), ctx)

		_, err = runner.Bootstrap(t.Context())
		require.NoError(t, err)
	})

	t.Run("idempotent", func(t *testing.T) {
		shutdownErr := errors.New("shutdown failed")
		var calls int
		runner := New(WithProviders(&testProvider{
			shutdown: func(ctx context.Context) (context.Context, error) {
				calls++
				return ctx, shutdownErr
			},
		}))

		ctx, err := runner.Bootstrap(t.Context())
		require.NoError(t, err)
		_, err = runner.Shutdown(ctx)
		assert.ErrorIs(t, err, shutdownErr)
		_, err = runner.Shutdown(ctx)
		assert.ErrorIs(t, err, shutdownErr)
		assert.Equal(t, 1, calls)
	})
}

func TestRunnerRunAfterBootstrap(t *testing.T) {
	runner := New()
	ctx, err := runner.Bootstrap(t.Context())
	require.NoError(t, err)

	err = runner.Run(ctx, func(context.Context) error { return nil })
	assert.ErrorIs(t, err, ErrAlreadyStarted)
	_, err = runner.Shutdown(ctx)
	require.NoError(t, err)
}

func TestRunnerRunRollsBackStartedProviders(t *testing.T) {
	bootstrapErr := errors.New("bootstrap failed")
	rollbackErr := errors.New("rollback failed")
	var order []string

	providerA := &testProvider{
		bootstrap: func(ctx context.Context) (context.Context, error) {
			order = append(order, "bootstrap-a")
			return context.WithValue(ctx, contextKey("a"), "a"), nil
		},
		shutdown: func(ctx context.Context) (context.Context, error) {
			order = append(order, "shutdown-a")
			assert.Equal(t, "a", ctx.Value(contextKey("a")))
			return ctx, rollbackErr
		},
	}
	providerB := &testProvider{
		bootstrap: func(ctx context.Context) (context.Context, error) {
			order = append(order, "bootstrap-b")
			return ctx, bootstrapErr
		},
		shutdown: func(ctx context.Context) (context.Context, error) {
			order = append(order, "shutdown-b")
			return ctx, nil
		},
	}
	providerC := &testProvider{
		bootstrap: func(ctx context.Context) (context.Context, error) {
			order = append(order, "bootstrap-c")
			return ctx, nil
		},
	}

	runner := New(WithProviders(providerA, providerB, providerC))
	err := runner.Run(t.Context(), func(context.Context) error {
		order = append(order, "handler")
		return nil
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, bootstrapErr)
	assert.ErrorIs(t, err, rollbackErr)
	assert.Equal(t, []string{"bootstrap-a", "bootstrap-b", "shutdown-a"}, order)
}

func TestRunnerRunJoinsHandlerAndShutdownErrors(t *testing.T) {
	handlerErr := errors.New("handler failed")
	shutdownAErr := errors.New("shutdown a failed")
	shutdownBErr := errors.New("shutdown b failed")
	var order []string

	providerA := &testProvider{
		shutdown: func(ctx context.Context) (context.Context, error) {
			order = append(order, "shutdown-a")
			assert.Nil(t, ctx.Value(contextKey("failed-shutdown")))
			return ctx, shutdownAErr
		},
	}
	providerB := &testProvider{
		shutdown: func(ctx context.Context) (context.Context, error) {
			order = append(order, "shutdown-b")
			return context.WithValue(ctx, contextKey("failed-shutdown"), true), shutdownBErr
		},
	}

	err := New(WithProviders(providerA, providerB)).Run(
		t.Context(),
		func(context.Context) error { return handlerErr },
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, handlerErr)
	assert.ErrorIs(t, err, shutdownAErr)
	assert.ErrorIs(t, err, shutdownBErr)
	assert.Equal(t, []string{"shutdown-b", "shutdown-a"}, order)
}

func TestRunnerRunShutdownIgnoresRuntimeCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	provider := &testProvider{
		bootstrap: func(ctx context.Context) (context.Context, error) {
			return context.WithValue(ctx, contextKey("value"), "value"), nil
		},
		shutdown: func(ctx context.Context) (context.Context, error) {
			assert.NoError(t, ctx.Err())
			assert.Equal(t, "value", ctx.Value(contextKey("value")))
			return ctx, nil
		},
	}

	err := New(WithProviders(provider)).Run(ctx, func(ctx context.Context) error {
		cancel()
		return ctx.Err()
	})

	assert.ErrorIs(t, err, context.Canceled)
}

func TestRunnerRunShutdownTimeout(t *testing.T) {
	provider := &testProvider{
		shutdown: func(ctx context.Context) (context.Context, error) {
			<-ctx.Done()
			return ctx, ctx.Err()
		},
	}

	err := New(
		WithProviders(provider),
		WithShutdownTimeout(time.Millisecond),
	).Run(t.Context(), func(context.Context) error { return nil })

	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestRunnerRunOnlyOnce(t *testing.T) {
	runner := New()
	require.NoError(t, runner.Run(t.Context(), func(context.Context) error { return nil }))
	assert.ErrorIs(t, runner.Run(t.Context(), func(context.Context) error { return nil }), ErrAlreadyStarted)
}

func TestRunnerRunConcurrentlyOnlyOnce(t *testing.T) {
	runner := New()
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)

	go func() {
		done <- runner.Run(t.Context(), func(context.Context) error {
			close(started)
			<-release
			return nil
		})
	}()
	<-started

	assert.ErrorIs(t, runner.Run(t.Context(), func(context.Context) error { return nil }), ErrAlreadyStarted)
	close(release)
	assert.NoError(t, <-done)
}

func TestRunnerRunOptionalArguments(t *testing.T) {
	t.Run("nil context", func(t *testing.T) {
		err := New().Run(nil, func(ctx context.Context) error { //nolint:staticcheck // Verify nil context handling.
			assert.NotNil(t, ctx)
			assert.NoError(t, ctx.Err())
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("nil handler", func(t *testing.T) {
		var bootstrapped, shutdown bool
		provider := &testProvider{
			bootstrap: func(ctx context.Context) (context.Context, error) {
				bootstrapped = true
				return ctx, nil
			},
			shutdown: func(ctx context.Context) (context.Context, error) {
				shutdown = true
				return ctx, nil
			},
		}

		err := New(WithProviders(provider)).Run(t.Context(), nil)
		require.NoError(t, err)
		assert.True(t, bootstrapped)
		assert.True(t, shutdown)
	})
}

func TestRunnerRunNilProviderContext(t *testing.T) {
	t.Run("bootstrap", func(t *testing.T) {
		provider := &testProvider{
			bootstrap: func(context.Context) (context.Context, error) {
				return nil, nil
			},
		}
		err := New(WithProviders(provider)).Run(
			t.Context(),
			func(context.Context) error { return nil },
		)
		assert.ErrorContains(t, err, "returned a nil context")
	})

	t.Run("shutdown", func(t *testing.T) {
		var shutdownA bool
		providerA := &testProvider{
			shutdown: func(ctx context.Context) (context.Context, error) {
				shutdownA = true
				return ctx, nil
			},
		}
		providerB := &testProvider{
			shutdown: func(context.Context) (context.Context, error) {
				return nil, nil
			},
		}
		err := New(WithProviders(providerA, providerB)).Run(
			t.Context(),
			func(context.Context) error { return nil },
		)
		assert.ErrorContains(t, err, "returned a nil context")
		assert.True(t, shutdownA)
	})
}

func TestRunnerRunShutdownsAfterPanic(t *testing.T) {
	var shutdown bool
	runner := New(WithProviders(&testProvider{
		shutdown: func(ctx context.Context) (context.Context, error) {
			shutdown = true
			return ctx, nil
		},
	}))

	assert.PanicsWithValue(t, "panic", func() {
		_ = runner.Run(t.Context(), func(context.Context) error {
			panic("panic")
		})
	})
	assert.True(t, shutdown)
}

func TestRunnerOptions(t *testing.T) {
	assert.NotPanics(t, func() {
		_ = New(
			nil,
			WithProviders(nil),
			WithShutdownTimeout(0),
		)
	})

	c := newConfig(
		WithProviders(nil),
		WithShutdownTimeout(time.Second),
		WithShutdownTimeout(0),
	)
	assert.Empty(t, c.providers)
	assert.Equal(t, time.Second, c.shutdownTimeout)
}

func TestRunnerRunRace(t *testing.T) {
	runner := New()
	start := make(chan struct{})
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Go(func() {
			<-start
			results <- runner.Run(t.Context(), func(context.Context) error { return nil })
		})
	}
	close(start)
	wg.Wait()
	close(results)

	var succeeded, alreadyStarted int
	for err := range results {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrAlreadyStarted):
			alreadyStarted++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	assert.Equal(t, 1, succeeded)
	assert.Equal(t, 1, alreadyStarted)
}
