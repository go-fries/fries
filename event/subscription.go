package event

import (
	"reflect"
	"slices"
	"sync/atomic"
)

// Subscription represents the handler registrations created by one
// [Dispatcher.Subscribe] call.
type Subscription struct {
	dispatcher    *Dispatcher
	registrations []registration
	active        atomic.Bool
}

type registration struct {
	typeOf reflect.Type
	entry  *listenerEntry
}

// Unsubscribe removes every registration owned by the Subscription. It reports
// whether at least one registration was removed. Unsubscribe is idempotent and
// returns false for a nil or inactive Subscription.
func (s *Subscription) Unsubscribe() bool {
	if s == nil || !s.active.CompareAndSwap(true, false) {
		return false
	}

	d := s.dispatcher
	d.mu.Lock()
	defer d.mu.Unlock()

	removed := false
	for _, registration := range s.registrations {
		entries := d.listeners[registration.typeOf]
		for i, entry := range entries {
			if entry != registration.entry {
				continue
			}
			entries = slices.Delete(entries, i, i+1)
			removed = true
			break
		}
		if len(entries) == 0 {
			delete(d.listeners, registration.typeOf)
		} else {
			d.listeners[registration.typeOf] = entries
		}
	}
	return removed
}
