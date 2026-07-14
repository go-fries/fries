package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"
)

// ErrAlreadyStarted is returned when a [Runner] has already started.
var ErrAlreadyStarted = errors.New("lifecycle: runner already started")

// Handler is the application work executed after all providers bootstrap.
type Handler func(context.Context) error

type runnerState uint8

const (
	runnerStateNew runnerState = iota
	runnerStateStarting
	runnerStateStarted
	runnerStateStopping
	runnerStateStopped
)

// Runner coordinates provider startup, application execution, and provider
// shutdown. A Runner may bootstrap only once.
//
// Runner implements [Provider], so it may be composed with another Runner.
type Runner struct {
	providers       []Provider
	shutdownTimeout time.Duration

	mu          sync.Mutex
	state       runnerState
	started     []Provider
	startDone   chan struct{}
	stopDone    chan struct{}
	shutdownErr error
}

// New creates a [Runner] configured by options.
func New(options ...Option) *Runner {
	c := newConfig(options...)
	return &Runner{
		providers:       c.providers,
		shutdownTimeout: c.shutdownTimeout,
	}
}

// Bootstrap starts providers in registration order and passes each returned
// context to the next provider.
//
// If startup fails, Bootstrap shuts down the providers that started
// successfully before returning the joined startup and rollback errors. A nil
// ctx is treated as [context.Background]. Bootstrap returns
// [ErrAlreadyStarted] after the first call, including while the first call is
// still running.
func (r *Runner) Bootstrap(ctx context.Context) (context.Context, error) {
	ctx = normalizeContext(ctx)

	r.mu.Lock()
	if r.state != runnerStateNew {
		r.mu.Unlock()
		return ctx, ErrAlreadyStarted
	}
	r.state = runnerStateStarting
	r.startDone = make(chan struct{})
	r.mu.Unlock()

	runtimeCtx, started, err := r.bootstrap(ctx)
	if err != nil {
		rollbackCtx, rollbackErr := r.withShutdownTimeout(
			runtimeCtx,
			func(ctx context.Context) (context.Context, error) {
				return r.shutdown(ctx, started)
			},
		)

		r.mu.Lock()
		r.state = runnerStateStopped
		r.shutdownErr = rollbackErr
		close(r.startDone)
		r.mu.Unlock()

		return rollbackCtx, errors.Join(err, rollbackErr)
	}

	r.mu.Lock()
	r.started = started
	r.state = runnerStateStarted
	close(r.startDone)
	r.mu.Unlock()

	return runtimeCtx, nil
}

// Shutdown stops successfully bootstrapped providers in reverse order and
// passes each returned context to the next provider.
//
// Shutdown uses ctx as supplied, allowing its caller to control cancellation
// and deadlines. A nil ctx is treated as [context.Background]. Calls made
// before Bootstrap are no-ops. Once shutdown starts, later calls wait for it to
// finish and return its error without running providers again.
func (r *Runner) Shutdown(ctx context.Context) (context.Context, error) {
	ctx = normalizeContext(ctx)

	for {
		r.mu.Lock()
		switch r.state {
		case runnerStateNew:
			r.mu.Unlock()
			return ctx, nil
		case runnerStateStarting:
			done := r.startDone
			r.mu.Unlock()
			<-done
		case runnerStateStarted:
			providers := slices.Clone(r.started)
			r.state = runnerStateStopping
			r.stopDone = make(chan struct{})
			r.mu.Unlock()

			shutdownCtx, err := r.shutdown(ctx, providers)

			r.mu.Lock()
			r.started = nil
			r.shutdownErr = err
			r.state = runnerStateStopped
			close(r.stopDone)
			r.mu.Unlock()

			return shutdownCtx, err
		case runnerStateStopping:
			done := r.stopDone
			r.mu.Unlock()
			<-done
		case runnerStateStopped:
			err := r.shutdownErr
			r.mu.Unlock()
			return ctx, err
		default:
			r.mu.Unlock()
			panic("lifecycle: invalid runner state")
		}
	}
}

// Run bootstraps providers, executes handler, and shuts down every provider
// that bootstrapped successfully. Provider shutdown runs in reverse order.
//
// Run delegates startup and shutdown to [Runner.Bootstrap] and
// [Runner.Shutdown]. It gives shutdown a timeout context derived from the final
// runtime context with its cancellation and deadline removed, preserving values
// added during bootstrap. Handler and shutdown errors are joined. Run returns
// [ErrAlreadyStarted] if the Runner has already started.
//
// A nil ctx is treated as [context.Background]. A nil handler is treated as a
// no-op, so the runner bootstraps and immediately shuts down its providers.
func (r *Runner) Run(ctx context.Context, handler Handler) (err error) {
	runtimeCtx, err := r.Bootstrap(ctx)
	if err != nil {
		return err
	}

	defer func() {
		_, shutdownErr := r.withShutdownTimeout(runtimeCtx, r.Shutdown)
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

func (r *Runner) withShutdownTimeout(
	ctx context.Context,
	shutdown func(context.Context) (context.Context, error),
) (context.Context, error) {
	ctx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		r.shutdownTimeout,
	)
	defer cancel()

	return shutdown(ctx)
}

func (r *Runner) shutdown(
	ctx context.Context,
	providers []Provider,
) (context.Context, error) {
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
	return ctx, errors.Join(errs...)
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func providerError(phase string, provider Provider, err error) error {
	return fmt.Errorf("lifecycle: %s provider %T: %w", phase, provider, err)
}

func nilContextError(phase string, provider Provider) error {
	return fmt.Errorf("lifecycle: %s provider %T returned a nil context", phase, provider)
}
