package event

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"
)

type Dispatcher struct {
	mu             sync.RWMutex
	listeners      []AnyListener
	middleware     []Middleware
	wg             sync.WaitGroup
	option         *options
	runningOptions *runningOptions
}

type options struct {
	// [parallel] limits the number of active goroutines in listeners to at most n.
	// A negative value indicates no limit. A limit of zero will prevent any new goroutines from being added.
	// Any subsequent call to the listener will block until it can add an active goroutine without exceeding the configured limit.
	// The limit must not be modified while any listener in the listeners are active.
	parallel int

	// [withError] enforces an error interrupt. When [withError] is true, one of the listeners returns an error,
	// which interrupts the execution of other listeners in the listener collection
	// Note: When parallel is set to an integer of -1 or>1 and an error interrupt is thrown,
	// if there is a blocking operation in the listener, the listener implementation should actively check ctx.Done()
	// to ensure proper cancellation and avoid interruption failures
	withError bool
}

type Option func(option *options)

func WithError() func(option *options) {
	return func(option *options) {
		option.withError = true
	}
}

func WithoutError() func(option *options) {
	return func(option *options) {
		option.withError = false
	}
}

func WithParallel(parallel int) func(option *options) {
	return func(option *options) {
		option.parallel = parallel
	}
}

type runningOptions struct {
	options
	// others running option...
}

type RunningOption func(option *runningOptions)

func WithRunningError() func(option *runningOptions) {
	return func(option *runningOptions) {
		option.withError = true
	}
}

func WithoutRunningError() func(option *runningOptions) {
	return func(option *runningOptions) {
		option.withError = false
	}
}

func WithRunningParallel(parallel int) func(option *runningOptions) {
	return func(option *runningOptions) {
		option.parallel = parallel
	}
}

func NewDispatcher(opts ...Option) *Dispatcher {
	defaultOption := &options{parallel: -1}
	for _, opt := range opts {
		opt(defaultOption)
	}
	return &Dispatcher{
		listeners:      make([]AnyListener, 0),
		option:         defaultOption,
		runningOptions: &runningOptions{*defaultOption},
	}
}

func (d *Dispatcher) Use(mws ...Middleware) {
	d.middleware = append(d.middleware, mws...)
}

func (d *Dispatcher) RegisterListeners(ls ...AnyListener) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.listeners = append(d.listeners, ls...)
}

// Dispatch the event to listeners
// If the options of the dispatch method have a value,
// it will overwrite the options of the NewDispatch method
func (d *Dispatcher) Dispatch(ctx context.Context, event any, options ...RunningOption) error {
	currentOption := *d.runningOptions
	for _, option := range options {
		option(&currentOption)
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(currentOption.parallel)

	middleChain := Chain(d.middleware...)
	for _, l := range d.listeners {
		d.wg.Add(1)
		eg.Go(func() error {
			defer d.wg.Done()

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				handler := middleChain(func(ctx context.Context, event any) error {
					return l.Handle(ctx, event)
				})

				if err := handler(ctx, event); err != nil && currentOption.withError {
					return err
				}
				return nil
			}
		})
	}
	return eg.Wait()
}

func (d *Dispatcher) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.listeners = make([]AnyListener, 0)
}

func (d *Dispatcher) Wait() {
	d.wg.Wait()
}
