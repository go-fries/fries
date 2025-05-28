package eventdispatcher

import (
	"context"
	"fmt"
	"sync"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"golang.org/x/sync/errgroup"
)

var (
	ErrNoListener = fmt.Errorf("no listener registered for the event")
)

type Dispatcher struct {
	listeners map[string][]Listener
	rw        sync.RWMutex
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		listeners: make(map[string][]Listener),
	}
}

func (d *Dispatcher) AddListener(typed string, listener Listener) {
	d.rw.Lock()
	defer d.rw.Unlock()

	if _, exists := d.listeners[typed]; !exists {
		d.listeners[typed] = make([]Listener, 0)
	}

	d.listeners[typed] = append(d.listeners[typed], listener)
}

func (d *Dispatcher) Dispatch(ctx context.Context, event cloudevents.Event) error {
	d.rw.RLock()
	defer d.rw.RUnlock()

	for typed, listener := range d.listeners {
		if event.Type() == typed {
			eg, ctx := errgroup.WithContext(NewContext(ctx, event))
			for _, l := range listener {
				eg.Go(func() error {
					return l.Handle(ctx, event)
				})
			}
			return eg.Wait()
		}
	}

	return ErrNoListener
}

func (d *Dispatcher) Reset() {
	d.rw.Lock()
	defer d.rw.Unlock()

	d.listeners = make(map[string][]Listener)
}
