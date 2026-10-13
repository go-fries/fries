package signal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"runtime/debug"
	"sync"
)

var (
	// ErrAlreadyStarted means Run was called on a Router that has already
	// started or has been closed by Shutdown.
	ErrAlreadyStarted = errors.New("signal: router already started or closed")
	// ErrRegistrationClosed means registration was attempted after Run or
	// Shutdown froze the Router's configuration.
	ErrRegistrationClosed = errors.New("signal: registration closed")
	// ErrInvalidHandler means a handler or function is nil, including a typed nil.
	ErrInvalidHandler = errors.New("signal: invalid handler")
	// ErrInvalidSignal means no signals were supplied, or a signal is nil or
	// cannot be used as a routing key. Host-specific support is determined by
	// the standard library's os/signal package.
	ErrInvalidSignal = errors.New("signal: invalid signal")
)

type routerState uint8

const (
	routerNew routerState = iota
	routerRunning
	routerStopping
	routerStopped
)

type registration struct {
	id      int
	handler Handler
	signals []os.Signal
	pending chan os.Signal
}

// Router routes explicitly registered operating system signals. Each
// registration has one worker and one pending notification slot. A registration
// runs serially; different registrations may run concurrently. When its pending
// slot is full, the new notification is dropped and a warning is logged.
//
// A Router runs only once. Register handlers before Run. Run and Shutdown may be
// called concurrently. A Router must be constructed with New and must not be
// copied. It never exits the process or resets another subscriber's signals.
type Router struct {
	config config
	source signalSource

	mu            sync.Mutex
	state         routerState
	registrations []*registration
	stop          chan struct{}
	done          chan struct{}
	cancel        context.CancelFunc
	abortErr      error
	runErr        error
}

// New creates a Router without subscribing to any signals. Nil options are
// ignored. Signal subscriptions are installed by Run and removed before it
// waits for active handlers to finish.
func New(opts ...Option) *Router {
	return &Router{
		config: newConfig(opts...),
		source: osSignalSource{},
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
}

// Listen registers a function for sig through the same path as Register.
func (r *Router) Listen(sig os.Signal, fn func(context.Context, os.Signal) error) error {
	return r.Register(HandlerFunc(fn), sig)
}

// Register adds one independent registration for handler and signals. Duplicate
// signals within this call are collapsed. Repeated calls create independent
// registrations, even for the same handler object; such an object must support
// concurrent calls. Signals are copied and no registration is added on error.
//
// Signals must be supported by os/signal on the host. Register rejects missing,
// nil, and non-comparable signals, but does not validate platform signal numbers
// or whether a signal can be caught. It returns ErrRegistrationClosed after Run
// or Shutdown, ErrInvalidHandler for a nil handler, or ErrInvalidSignal for an
// invalid routing key.
func (r *Router) Register(handler Handler, signals ...os.Signal) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.state != routerNew {
		return ErrRegistrationClosed
	}
	if isNil(handler) {
		return ErrInvalidHandler
	}
	if len(signals) == 0 {
		return ErrInvalidSignal
	}

	unique := make([]os.Signal, 0, len(signals))
	seen := make(map[os.Signal]struct{}, len(signals))
	for i, sig := range signals {
		if isNil(sig) || !reflect.ValueOf(sig).Comparable() {
			return fmt.Errorf("%w at index %d", ErrInvalidSignal, i)
		}
		if _, ok := seen[sig]; !ok {
			seen[sig] = struct{}{}
			unique = append(unique, sig)
		}
	}

	r.registrations = append(r.registrations, &registration{
		id:      len(r.registrations) + 1,
		handler: handler,
		signals: unique,
		pending: make(chan os.Signal, 1),
	})
	return nil
}

// Run installs signal subscriptions and blocks until reception has stopped and
// all workers have exited. An empty Router waits without subscribing to signals.
// Run returns ErrAlreadyStarted on a second call, including concurrent calls or
// calls after Shutdown. A context canceled before Run consumes the instance
// without installing subscriptions.
//
// Handlers receive a context derived from ctx. Cancellation of ctx cancels active
// handlers and discards pending notifications. Shutdown instead waits for active
// handlers without canceling them, unless a shutdown caller's context expires.
// Run returns the joined context and shutdown-abort errors, or nil after graceful
// shutdown. Handler errors and panics are logged, not returned by Run.
//
// Run cannot return until a handler that ignores cancellation actually exits.
// Use Shutdown with a deadline to bound the caller's wait, not the handler's life.
func (r *Router) Run(ctx context.Context) error {
	r.mu.Lock()
	if r.state != routerNew {
		r.mu.Unlock()
		return ErrAlreadyStarted
	}
	handlerCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	r.state = routerRunning
	r.mu.Unlock()
	defer cancel()

	r.config.logger.InfoContext(ctx, "[Signal] router starting")
	routes, signals := r.routes()
	incoming := make(chan os.Signal, max(1, len(signals)))
	subscribed := ctx.Err() == nil && len(signals) > 0
	if subscribed {
		r.source.Notify(incoming, signals...)
	}

	var workers sync.WaitGroup
	for _, reg := range r.registrations {
		workers.Go(func() { r.work(handlerCtx, reg) })
	}
	r.receive(handlerCtx, incoming, routes)
	r.beginShutdown()
	// Unregister before waiting: a stuck handler must not retain OS subscriptions.
	if subscribed {
		r.source.Stop(incoming)
	}
	for _, reg := range r.registrations {
		select {
		case <-reg.pending:
		default:
		}
	}
	workers.Wait()

	r.mu.Lock()
	r.runErr = errors.Join(ctx.Err(), r.abortErr)
	r.state = routerStopped
	close(r.done)
	err := r.runErr
	r.mu.Unlock()
	return err
}

// Shutdown stops reception and admission of new handler calls, discards pending
// notifications, and waits for active handlers and signal cleanup. Calling it
// before Run permanently closes the Router without subscribing to signals.
//
// If ctx is canceled before completion, Shutdown cancels the handler context
// and returns ctx.Err. The first such error also becomes part of Run's result.
// Any concurrent shutdown caller can trigger this cancellation. A timed-out
// Router remains stopping until its workers actually exit. Later calls can wait
// again and return the final Run result; completed shutdown takes precedence over
// an already-canceled ctx. Concurrent callers do not repeat cleanup.
//
// Do not call Shutdown from a handler: it would wait for that handler itself.
// To request application exit from a handler, cancel the application's context.
func (r *Router) Shutdown(ctx context.Context) error {
	r.beginShutdown()
	select {
	case <-r.done:
		return r.result()
	default:
	}

	select {
	case <-r.done:
		return r.result()
	case <-ctx.Done():
		r.mu.Lock()
		if r.state == routerStopped {
			err := r.runErr
			r.mu.Unlock()
			return err
		}
		if r.abortErr == nil {
			r.abortErr = ctx.Err()
		}
		r.cancel()
		r.mu.Unlock()
		return ctx.Err()
	}
}

func (r *Router) beginShutdown() {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch r.state {
	case routerNew:
		r.state = routerStopped
		close(r.stop)
		close(r.done)
	case routerRunning:
		r.state = routerStopping
		close(r.stop)
	case routerStopping, routerStopped:
	}
}

func (r *Router) result() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.runErr
}

func (r *Router) routes() (map[os.Signal][]*registration, []os.Signal) {
	routes := make(map[os.Signal][]*registration)
	var signals []os.Signal
	for _, reg := range r.registrations {
		for _, sig := range reg.signals {
			if _, ok := routes[sig]; !ok {
				signals = append(signals, sig)
			}
			routes[sig] = append(routes[sig], reg)
		}
	}
	return routes, signals
}

func (r *Router) receive(ctx context.Context, incoming <-chan os.Signal, routes map[os.Signal][]*registration) {
	for {
		if ctx.Err() != nil {
			return
		}
		select {
		case <-r.stop:
			return
		default:
		}
		select {
		case <-ctx.Done():
			return
		case <-r.stop:
			return
		case sig, ok := <-incoming:
			if !ok {
				return
			}
			for _, reg := range routes[sig] {
				r.offer(ctx, reg, sig)
			}
		}
	}
}

func (r *Router) offer(ctx context.Context, reg *registration, sig os.Signal) {
	r.mu.Lock()
	if r.state != routerRunning || ctx.Err() != nil {
		r.mu.Unlock()
		return
	}
	select {
	case reg.pending <- sig:
		r.mu.Unlock()
	default:
		r.mu.Unlock()
		r.config.logger.WarnContext(ctx, "[Signal] notification dropped",
			slog.String("signal", sig.String()), slog.Int("registration", reg.id))
	}
}

func (r *Router) work(ctx context.Context, reg *registration) {
	for {
		select {
		case <-r.stop:
			return
		case sig := <-reg.pending:
			r.mu.Lock()
			admitted := r.state == routerRunning && ctx.Err() == nil
			r.mu.Unlock()
			if !admitted {
				return
			}
			r.handle(ctx, reg, sig)
		}
	}
}

func (r *Router) handle(ctx context.Context, reg *registration, sig os.Signal) {
	defer func() {
		if recovered := recover(); recovered != nil {
			r.config.logger.ErrorContext(ctx, "[Signal] handler panic",
				slog.String("signal", sig.String()), slog.Int("registration", reg.id),
				slog.Any("panic", recovered), slog.String("stack", string(debug.Stack())))
		}
	}()
	if err := reg.handler.Handle(ctx, sig); err != nil {
		r.config.logger.ErrorContext(ctx, "[Signal] handler failed",
			slog.String("signal", sig.String()), slog.Int("registration", reg.id), slog.Any("error", err))
	}
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
