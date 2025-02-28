package event

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"
)

type Dispatcher struct {
	mu        sync.RWMutex
	listeners []AnyListener
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		listeners: make([]AnyListener, 0),
	}
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
			return l.Handle(ctx, event)
		})
	}
	return eg.Wait()
}

func (d *Dispatcher) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.listeners = make([]AnyListener, 0)
}
