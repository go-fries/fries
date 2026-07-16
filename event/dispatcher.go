package event

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
)

var (
	// ErrInvalidContext is returned when Dispatch receives a nil context.
	ErrInvalidContext = errors.New("event: invalid context")
	// ErrNilEvent is returned when Dispatch receives an untyped nil event.
	ErrNilEvent = errors.New("event: nil event")
)

// Dispatcher synchronously dispatches in-process events to handlers registered
// for their exact concrete types. A Dispatcher is safe for concurrent use.
type Dispatcher struct {
	mu         sync.RWMutex
	listeners  map[reflect.Type][]*listenerEntry
	middleware []Middleware
}

type listenerEntry struct {
	typeOf reflect.Type
	next   Next
}

// New creates a Dispatcher configured by options.
func New(options ...Option) *Dispatcher {
	c := newConfig(options...)
	return &Dispatcher{
		listeners:  make(map[reflect.Type][]*listenerEntry),
		middleware: c.middleware,
	}
}

// Subscribe registers listeners for their exact event types. The returned
// Subscription owns every registration created by this call.
//
// Subscribe panics if the Dispatcher or any listener is nil. Calling Subscribe
// without listeners returns an inactive Subscription.
func (d *Dispatcher) Subscribe(listeners ...Listener) *Subscription {
	if d == nil {
		panic("event: nil dispatcher")
	}

	subscription := &Subscription{dispatcher: d}
	if len(listeners) == 0 {
		return subscription
	}

	entries := make([]*listenerEntry, len(listeners))
	for i, listener := range listeners {
		if listener == nil {
			panic("event: nil listener")
		}

		definition := listener.definition()
		next := Chain(d.middleware...)(definition.next)
		if next == nil {
			panic("event: middleware returned a nil next function")
		}
		entries[i] = &listenerEntry{typeOf: definition.typeOf, next: next}
	}

	d.mu.Lock()
	for _, entry := range entries {
		d.listeners[entry.typeOf] = append(d.listeners[entry.typeOf], entry)
		subscription.registrations = append(subscription.registrations, registration{
			typeOf: entry.typeOf,
			entry:  entry,
		})
	}
	d.mu.Unlock()

	subscription.active.Store(true)
	return subscription
}

// Dispatch synchronously dispatches value to handlers registered for its exact
// concrete type. Handlers execute serially and stop at the first error by
// default. Dispatch options can enable bounded concurrency or continued error
// collection for this call.
//
// Dispatch returns [ErrInvalidContext] for a nil context and [ErrNilEvent] for an
// untyped nil event. It panics if called on a nil Dispatcher.
func (d *Dispatcher) Dispatch(
	ctx context.Context,
	value any,
	options ...DispatchOption,
) error {
	if d == nil {
		panic("event: nil dispatcher")
	}
	if ctx == nil {
		return ErrInvalidContext
	}
	if value == nil {
		return ErrNilEvent
	}
	if cause := context.Cause(ctx); cause != nil {
		return cause
	}

	c := newDispatchConfig(options...)
	entries := d.snapshot(reflect.TypeOf(value))
	if len(entries) == 0 {
		return nil
	}
	if c.concurrency == 1 || len(entries) == 1 {
		return dispatchSerial(ctx, value, entries, c.continueOnError)
	}
	return dispatchConcurrent(ctx, value, entries, c)
}

func (d *Dispatcher) snapshot(typeOf reflect.Type) []*listenerEntry {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return append([]*listenerEntry(nil), d.listeners[typeOf]...)
}

func dispatchSerial(
	ctx context.Context,
	value any,
	entries []*listenerEntry,
	continueOnError bool,
) error {
	var errs []error
	for _, entry := range entries {
		if cause := context.Cause(ctx); cause != nil {
			return joinErrors(appendContextCause(errs, cause))
		}

		err := entry.next(ctx, value)
		if err != nil {
			errs = append(errs, err)
			if !continueOnError {
				return joinErrors(appendContextCause(errs, context.Cause(ctx)))
			}
		}

		if cause := context.Cause(ctx); cause != nil {
			return joinErrors(appendContextCause(errs, cause))
		}
	}
	return joinErrors(errs)
}

func dispatchConcurrent(
	ctx context.Context,
	value any,
	entries []*listenerEntry,
	c dispatchConfig,
) error {
	workerCount := min(c.concurrency, len(entries))
	results := make([]error, len(entries))

	runCtx := ctx
	cancel := func(error) {}
	if !c.continueOnError {
		runCtx, cancel = context.WithCancelCause(ctx)
	}
	defer cancel(nil)

	var (
		nextIndex  atomic.Int64
		firstError error
		firstOnce  sync.Once
		workers    sync.WaitGroup
	)
	nextIndex.Store(-1)

	for range workerCount {
		workers.Go(func() {
			for {
				if context.Cause(runCtx) != nil {
					return
				}

				index := int(nextIndex.Add(1))
				if index >= len(entries) || context.Cause(runCtx) != nil {
					return
				}

				err := entries[index].next(runCtx, value)
				if err == nil {
					continue
				}
				if c.continueOnError {
					results[index] = err
					continue
				}

				firstOnce.Do(func() {
					firstError = err
					cancel(err)
				})
				return
			}
		})
	}
	workers.Wait()

	if !c.continueOnError {
		return joinErrors(appendContextCause([]error{firstError}, context.Cause(ctx)))
	}

	errs := make([]error, 0, len(results)+1)
	for _, err := range results {
		if err != nil {
			errs = append(errs, err)
		}
	}
	return joinErrors(appendContextCause(errs, context.Cause(ctx)))
}

func appendContextCause(errs []error, cause error) []error {
	if cause == nil {
		return compactErrors(errs)
	}
	for _, err := range errs {
		if err != nil && errors.Is(err, cause) {
			return compactErrors(errs)
		}
	}
	return append(compactErrors(errs), cause)
}

func compactErrors(errs []error) []error {
	compacted := errs[:0]
	for _, err := range errs {
		if err != nil {
			compacted = append(compacted, err)
		}
	}
	return compacted
}

func joinErrors(errs []error) error {
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		return errors.Join(errs...)
	}
}
