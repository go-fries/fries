package health

import (
	"context"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

// Registry stores named health checks and runs a stable snapshot of them.
// A Registry is safe for concurrent registration and checking.
type Registry struct {
	mu          sync.RWMutex
	checks      []checkEntry
	names       map[string]struct{}
	timeout     time.Duration
	concurrency int
}

type checkEntry struct {
	name    string
	checker Checker
}

// New creates a [Registry] configured by options.
func New(options ...Option) *Registry {
	c := newConfig(options...)
	return &Registry{
		names:       make(map[string]struct{}),
		timeout:     c.timeout,
		concurrency: c.concurrency,
	}
}

// Register adds a named checker. Checks appear in reports in registration
// order.
//
// Register panics if the Registry is nil, name is empty, checker is nil, or
// name is already registered.
func (r *Registry) Register(name string, checker Checker) {
	if r == nil {
		panic("health: nil registry")
	}
	if name == "" {
		panic("health: empty check name")
	}
	if isNilChecker(checker) {
		panic("health: nil checker")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.names[name]; exists {
		panic("health: duplicate check name: " + name)
	}
	r.names[name] = struct{}{}
	r.checks = append(r.checks, checkEntry{name: name, checker: checker})
}

// Check runs a snapshot of the registered checks. Checks may run concurrently,
// but results remain in registration order. Ordinary check errors do not stop
// other checks.
//
// Check panics if the Registry or ctx is nil.
func (r *Registry) Check(ctx context.Context) Report {
	if r == nil {
		panic("health: nil registry")
	}
	if ctx == nil {
		panic("health: nil context")
	}

	startedAt := time.Now()
	entries := r.snapshot()
	results := make([]Result, len(entries))
	for i, entry := range entries {
		results[i].Name = entry.name
	}

	if len(entries) > 0 {
		r.run(ctx, entries, results)
	}
	return Report{
		StartedAt: startedAt,
		Duration:  time.Since(startedAt),
		Results:   results,
	}
}

func (r *Registry) snapshot() []checkEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return append([]checkEntry(nil), r.checks...)
}

func (r *Registry) run(
	ctx context.Context,
	entries []checkEntry,
	results []Result,
) {
	runCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	workerCount := min(r.concurrency, len(entries))
	var (
		next    atomic.Int64
		workers sync.WaitGroup
	)
	next.Store(-1)

	for range workerCount {
		workers.Go(func() {
			for {
				index := int(next.Add(1))
				if index >= len(entries) {
					return
				}

				if cause := context.Cause(runCtx); cause != nil {
					results[index].Err = cause
					continue
				}

				startedAt := time.Now()
				err := entries[index].checker.Check(runCtx)
				results[index].Duration = time.Since(startedAt)
				if err == nil {
					err = context.Cause(runCtx)
				}
				results[index].Err = err
			}
		})
	}
	workers.Wait()
}

func isNilChecker(checker Checker) bool {
	if checker == nil {
		return true
	}
	value := reflect.ValueOf(checker)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
