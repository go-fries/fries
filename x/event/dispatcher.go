package event

import (
	"sync"
)

type Event interface {
	Event() any
}

type Listener interface {
	Listen() []Event
	Handle(event Event)
}

type Dispatcher struct {
	listeners map[any][]Listener
	mu        sync.RWMutex
	recovery  func(err any, listener Listener, event Event)
	waiter    sync.WaitGroup
}

type Option func(*Dispatcher)

func WithRecovery(recovery func(err any, listener Listener, event Event)) Option {
	return func(d *Dispatcher) {
		d.recovery = recovery
	}
}

func NewDispatcher(opts ...Option) *Dispatcher {
	d := &Dispatcher{
		listeners: make(map[any][]Listener),
	}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

func (d *Dispatcher) AddListener(listeners ...Listener) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, listener := range listeners {
		if listener == nil {
			continue
		}
		for _, event := range listener.Listen() {
			e := event.Event()
			if _, ok := d.listeners[e]; !ok {
				d.listeners[e] = make([]Listener, 0)
			}
			d.listeners[e] = append(d.listeners[e], listener)
		}
	}
}

func (d *Dispatcher) Dispatch(event Event) {
	if listeners, ok := d.listeners[event.Event()]; ok {
		for _, listener := range listeners {
			d.waiter.Add(1)
			d.handle(listener, event)
		}
	}
}

func (d *Dispatcher) DispatchAsync(event Event) {
	if listeners, ok := d.listeners[event.Event()]; ok {
		for _, listener := range listeners {
			d.waiter.Add(1)
			go d.handle(listener, event)
		}
	}
}

func (d *Dispatcher) handle(listener Listener, event Event) {
	defer d.waiter.Done()

	if d.recovery != nil {
		defer func() {
			if err := recover(); err != nil {
				d.recovery(err, listener, event)
			}
		}()
	}

	listener.Handle(event)
}

func (d *Dispatcher) Wait() {
	d.waiter.Wait()
}
