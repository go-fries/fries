package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync/atomic"
	"time"
)

// ErrAlreadyRun is returned when a [Runner] is run more than once.
var ErrAlreadyRun = errors.New("lifecycle: runner already run")

// Handler is the application work executed after all providers bootstrap.
type Handler func(context.Context) error

// Runner coordinates provider startup, application execution, and provider
// shutdown. A Runner may be run only once.
type Runner struct {
	providers       []Provider
	shutdownTimeout time.Duration
	ran             atomic.Bool
}

// New creates a [Runner] configured by options.
func New(options ...Option) *Runner {
	c := newConfig(options...)
	return &Runner{
		providers:       c.providers,
		shutdownTimeout: c.shutdownTimeout,
	}
}

// Run bootstraps providers, executes handler, and shuts down every provider
// that bootstrapped successfully. Provider shutdown runs in reverse order.
//
// If startup fails, Run rolls back the providers that started successfully.
// Shutdown uses a timeout context derived from the final runtime context with
// its cancellation and deadline removed, preserving values added during
// bootstrap. Handler and shutdown errors are joined. Run returns
// [ErrAlreadyRun] after its first invocation.
//
// A nil ctx is treated as [context.Background]. A nil handler is treated as a
// no-op, so the runner bootstraps and immediately shuts down its providers.
func (r *Runner) Run(ctx context.Context, handler Handler) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if !r.ran.CompareAndSwap(false, true) {
		return ErrAlreadyRun
	}

	runtimeCtx, started, err := r.bootstrap(ctx)
	if err != nil {
		shutdownErr := r.shutdown(runtimeCtx, started)
		return errors.Join(err, shutdownErr)
	}

	defer func() {
		shutdownErr := r.shutdown(runtimeCtx, started)
		err = errors.Join(err, shutdownErr)
	}()

	if handler != nil {
		return handler(runtimeCtx)
	}
	return nil
}

func (r *Runner) bootstrap(ctx context.Context) (context.Context, []Provider, error) {
	started := make([]Provider, 0, len(r.providers))
	for _, provider := range r.providers {
		next, err := provider.Bootstrap(ctx)
		if err != nil {
			return ctx, started, providerError("bootstrap", provider, err)
		}
		if next == nil {
			return ctx, started, nilContextError("bootstrap", provider)
		}
		ctx = next
		started = append(started, provider)
	}
	return ctx, started, nil
}

func (r *Runner) shutdown(ctx context.Context, providers []Provider) error {
	ctx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		r.shutdownTimeout,
	)
	defer cancel()

	var errs []error
	for _, provider := range slices.Backward(providers) {
		next, err := provider.Shutdown(ctx)
		if err != nil {
			errs = append(errs, providerError("shutdown", provider, err))
			continue
		}
		if next == nil {
			errs = append(errs, nilContextError("shutdown", provider))
			continue
		}
		ctx = next
	}
	return errors.Join(errs...)
}

func providerError(phase string, provider Provider, err error) error {
	return fmt.Errorf("lifecycle: %s provider %T: %w", phase, provider, err)
}

func nilContextError(phase string, provider Provider) error {
	return fmt.Errorf("lifecycle: %s provider %T returned a nil context", phase, provider)
}
