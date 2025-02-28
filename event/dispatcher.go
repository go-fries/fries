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
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		listeners: make([]AnyListener, 0),
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
	for _, l := range d.listeners {
		eg.Go(func() error {
			handler := Chain(d.middleware...)(func(ctx context.Context, event any) error {
				return l.Handle(ctx, event)
			})
			return handler(ctx, event)
		})
	}
	return eg.Wait()
}

func (d *Dispatcher) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.listeners = make([]AnyListener, 0)
}
