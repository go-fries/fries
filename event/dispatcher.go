package event

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"
)

type Dispatcher struct {
	mu         sync.RWMutex
	listeners  []AnyListener
	middleware []Middleware
	option     *Option
}

type Option struct {
	WithError bool
	Works     int
}

type Options func(option *Option)

func NewDispatcher(opts ...Options) *Dispatcher {
	defaultOption := &Option{
		WithError: true,
		Works:     1,
	}
	for _, opt := range opts {
		opt(defaultOption)
	}
	return &Dispatcher{
		listeners: make([]AnyListener, 0),
		option:    defaultOption,
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

func (d *Dispatcher) Dispatch(ctx context.Context, event any) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(d.option.Works)

	middleChain := Chain(d.middleware...)
	for _, l := range d.listeners {
		ll := l
		eg.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				handler := middleChain(func(ctx context.Context, event any) error {
					return ll.Handle(ctx, event)
				})

				if err := handler(ctx, event); err != nil && d.option.WithError {
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

// Wait keep compatible
func (d *Dispatcher) Wait() {}
