package signal

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"slices"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouter_RegisterValidatesAtomically(t *testing.T) {
	var nilHandler *objectHandler
	var nilSignal *pointerSignal
	var nilFunction HandlerFunc
	handler := HandlerFunc(func(context.Context, os.Signal) error { return nil })
	for _, tc := range []struct {
		name    string
		handler Handler
		signals []os.Signal
		wantErr error
	}{
		{name: "nil handler", signals: []os.Signal{os.Interrupt}, wantErr: ErrInvalidHandler},
		{name: "typed nil handler", handler: nilHandler, signals: []os.Signal{os.Interrupt}, wantErr: ErrInvalidHandler},
		{name: "nil function", handler: nilFunction, signals: []os.Signal{os.Interrupt}, wantErr: ErrInvalidHandler},
		{name: "no signals", handler: handler, wantErr: ErrInvalidSignal},
		{name: "nil signal", handler: handler, signals: []os.Signal{nil}, wantErr: ErrInvalidSignal},
		{name: "typed nil signal", handler: handler, signals: []os.Signal{nilSignal}, wantErr: ErrInvalidSignal},
		{name: "noncomparable signal", handler: handler, signals: []os.Signal{sliceSignal{1}}, wantErr: ErrInvalidSignal},
		{name: "invalid final signal", handler: handler, signals: []os.Signal{os.Interrupt, nil}, wantErr: ErrInvalidSignal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := quietRouter()
			require.ErrorIs(t, r.Register(tc.handler, tc.signals...), tc.wantErr)
			assert.Empty(t, r.registrations)
		})
	}
	assert.ErrorIs(t, quietRouter().Listen(os.Interrupt, nil), ErrInvalidHandler)
}

func TestRouter_RegisterCopiesAndDeduplicatesSignals(t *testing.T) {
	r := quietRouter()
	signals := []os.Signal{os.Interrupt, syscall.SIGTERM, os.Interrupt}
	handler := &objectHandler{}
	require.NoError(t, r.Register(handler, signals...))
	signals[0] = nil
	require.NoError(t, r.Register(handler, os.Interrupt))

	routes, subscribed := r.routes()
	assert.Equal(t, []os.Signal{os.Interrupt, syscall.SIGTERM}, subscribed)
	require.Len(t, r.registrations, 2)
	assert.Same(t, handler, r.registrations[0].handler)
	assert.Equal(t, []os.Signal{os.Interrupt, syscall.SIGTERM}, r.registrations[0].signals)
	assert.Len(t, routes[os.Interrupt], 2)
	assert.Len(t, routes[syscall.SIGTERM], 1)
	assert.Equal(t, 1, cap(r.registrations[0].pending))
}

func TestRouter_DispatchesFunctionsAndObjects(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := quietRouter()
		calls := make(chan os.Signal, 4)
		fn := func(_ context.Context, sig os.Signal) error { calls <- sig; return nil }
		require.NoError(t, r.Listen(os.Interrupt, fn))
		require.NoError(t, r.Register(&objectHandler{fn: fn}, os.Interrupt, syscall.SIGTERM))
		source, done := startRouter(t, t.Context(), r)
		assert.Equal(t, []os.Signal{os.Interrupt, syscall.SIGTERM}, source.signals)
		source.emit(os.Interrupt)
		synctest.Wait()
		require.Len(t, calls, 2)
		assert.Equal(t, os.Interrupt, <-calls)
		assert.Equal(t, os.Interrupt, <-calls)
		source.emit(syscall.SIGTERM)
		synctest.Wait()
		require.Len(t, calls, 1)
		assert.Equal(t, syscall.SIGTERM, <-calls)
		// An unrelated source notification must not call either handler.
		source.emit(syscall.SIGABRT)
		synctest.Wait()
		assert.Empty(t, calls)
		require.NoError(t, r.Shutdown(t.Context()))
		assert.NoError(t, <-done)
		assert.Equal(t, int32(1), source.stopCalls.Load())
	})
}

func TestRouter_RegistrationSerialAndOverflowDropsNewest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		logs := &recordingHandler{}
		r := New(WithLogger(slog.New(logs)))
		calls := make(chan os.Signal, 4)
		release := make(chan struct{})
		unblock := sync.OnceFunc(func() { close(release) })
		t.Cleanup(unblock)
		var active, maxActive atomic.Int32
		require.NoError(t, r.Register(HandlerFunc(func(_ context.Context, sig os.Signal) error {
			if n := active.Add(1); n > maxActive.Load() {
				maxActive.Store(n)
			}
			defer active.Add(-1)
			calls <- sig
			<-release
			return nil
		}), os.Interrupt, syscall.SIGTERM, syscall.SIGABRT))
		source, done := startRouter(t, t.Context(), r)
		source.emit(os.Interrupt)
		synctest.Wait()
		require.Len(t, calls, 1)
		assert.Equal(t, os.Interrupt, <-calls)
		source.emit(syscall.SIGTERM)
		synctest.Wait()
		source.emit(syscall.SIGABRT)
		synctest.Wait()
		assert.Empty(t, calls)
		warnings := logs.withMessage("[Signal] notification dropped")
		require.Len(t, warnings, 1)
		assert.Equal(t, slog.LevelWarn, warnings[0].Level)
		assert.Equal(t, map[string]any{"signal": syscall.SIGABRT.String(), "registration": int64(1)}, logAttrs(warnings[0]))
		unblock()
		synctest.Wait()
		require.Len(t, calls, 1)
		assert.Equal(t, syscall.SIGTERM, <-calls)
		assert.Equal(t, int32(1), maxActive.Load())
		require.NoError(t, r.Shutdown(t.Context()))
		assert.NoError(t, <-done)
	})
}

func TestRouter_SlowRegistrationDoesNotBlockOtherRegistrations(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := quietRouter()
		blocked := make(chan struct{})
		release := make(chan struct{})
		unblock := sync.OnceFunc(func() { close(release) })
		t.Cleanup(unblock)
		fast := make(chan os.Signal, 2)
		require.NoError(t, r.Listen(os.Interrupt, func(context.Context, os.Signal) error {
			close(blocked)
			<-release
			return nil
		}))
		require.NoError(t, r.Listen(os.Interrupt, func(_ context.Context, sig os.Signal) error { fast <- sig; return nil }))
		require.NoError(t, r.Listen(syscall.SIGTERM, func(_ context.Context, sig os.Signal) error { fast <- sig; return nil }))
		source, done := startRouter(t, t.Context(), r)
		source.emit(os.Interrupt)
		<-blocked
		synctest.Wait()
		require.Len(t, fast, 1)
		assert.Equal(t, os.Interrupt, <-fast)
		source.emit(syscall.SIGTERM)
		synctest.Wait()
		require.Len(t, fast, 1)
		assert.Equal(t, syscall.SIGTERM, <-fast)
		unblock()
		require.NoError(t, r.Shutdown(t.Context()))
		assert.NoError(t, <-done)
	})
}

func TestRouter_LogsErrorsAndPanicsAndContinues(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		logs := &recordingHandler{}
		r := New(WithLogger(slog.New(logs)))
		wantErr := errors.New("reload failed")
		var calls atomic.Int32
		require.NoError(t, r.Listen(os.Interrupt, func(context.Context, os.Signal) error {
			switch calls.Add(1) {
			case 1:
				return wantErr
			case 2:
				panic("reload panic")
			default:
				return nil
			}
		}))
		source, done := startRouter(t, t.Context(), r)
		for range 3 {
			source.emit(os.Interrupt)
			synctest.Wait()
		}
		assert.Equal(t, int32(3), calls.Load())
		failures := logs.withMessage("[Signal] handler failed")
		require.Len(t, failures, 1)
		assert.Equal(t, slog.LevelError, failures[0].Level)
		assert.Equal(t, map[string]any{"signal": os.Interrupt.String(), "registration": int64(1), "error": wantErr}, logAttrs(failures[0]))
		panics := logs.withMessage("[Signal] handler panic")
		require.Len(t, panics, 1)
		assert.Equal(t, slog.LevelError, panics[0].Level)
		attrs := logAttrs(panics[0])
		assert.Equal(t, "reload panic", attrs["panic"])
		assert.Contains(t, attrs["stack"], "TestRouter_LogsErrorsAndPanicsAndContinues")
		assert.Equal(t, os.Interrupt.String(), attrs["signal"])
		assert.Equal(t, int64(1), attrs["registration"])
		require.NoError(t, r.Shutdown(t.Context()))
		assert.NoError(t, <-done) // Handler failures are not lifecycle failures.
	})
}

func TestRouter_ShutdownWaitsWithoutCancelingAndDiscardsPending(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := quietRouter()
		started := make(chan context.Context, 1)
		release := make(chan struct{})
		unblock := sync.OnceFunc(func() { close(release) })
		t.Cleanup(unblock)
		var calls atomic.Int32
		require.NoError(t, r.Listen(os.Interrupt, func(ctx context.Context, _ os.Signal) error {
			calls.Add(1)
			started <- ctx
			<-release
			return ctx.Err()
		}))
		source, done := startRouter(t, t.Context(), r)
		source.emit(os.Interrupt)
		handlerCtx := <-started
		source.emit(os.Interrupt)
		synctest.Wait()
		shutdown := make(chan error, 1)
		go func() { shutdown <- r.Shutdown(t.Context()) }()
		synctest.Wait()
		assert.Empty(t, shutdown)
		assert.Empty(t, done)
		assert.NoError(t, handlerCtx.Err())
		assert.Equal(t, routerStopping, stateOf(r))
		assert.Equal(t, int32(1), source.stopCalls.Load())
		assert.Empty(t, r.registrations[0].pending)
		assert.ErrorIs(t, r.Listen(os.Interrupt, func(context.Context, os.Signal) error { return nil }), ErrRegistrationClosed)
		// Even a notification offered concurrently with shutdown cannot be admitted.
		r.offer(t.Context(), r.registrations[0], os.Interrupt)
		unblock()
		assert.NoError(t, <-shutdown)
		assert.NoError(t, <-done)
		assert.Equal(t, int32(1), calls.Load())
		assert.ErrorIs(t, handlerCtx.Err(), context.Canceled) // Lifetime ends after join.
		assert.Equal(t, routerStopped, stateOf(r))
	})
}

func TestRouter_ShutdownTimeoutCancelsButDoesNotPretendWorkerExited(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := quietRouter()
		started := make(chan context.Context, 1)
		release := make(chan struct{})
		unblock := sync.OnceFunc(func() { close(release) })
		t.Cleanup(unblock)
		require.NoError(t, r.Listen(os.Interrupt, func(ctx context.Context, _ os.Signal) error {
			started <- ctx
			<-release // Deliberately ignore cancellation.
			return nil
		}))
		source, done := startRouter(t, t.Context(), r)
		source.emit(os.Interrupt)
		handlerCtx := <-started
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		require.ErrorIs(t, r.Shutdown(ctx), context.DeadlineExceeded)
		assert.ErrorIs(t, handlerCtx.Err(), context.Canceled)
		assert.Equal(t, routerStopping, stateOf(r))
		assert.Empty(t, done)
		assert.Equal(t, int32(1), source.stopCalls.Load())
		unblock()
		assert.ErrorIs(t, <-done, context.DeadlineExceeded)
		assert.ErrorIs(t, r.Shutdown(t.Context()), context.DeadlineExceeded)
		assert.Equal(t, routerStopped, stateOf(r))
	})
}

func TestRouter_CanceledShutdownCancelsCooperativeHandler(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := quietRouter()
		started := make(chan struct{})
		require.NoError(t, r.Listen(os.Interrupt, func(ctx context.Context, _ os.Signal) error {
			close(started)
			<-ctx.Done()
			return ctx.Err()
		}))
		source, done := startRouter(t, t.Context(), r)
		source.emit(os.Interrupt)
		<-started
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		assert.ErrorIs(t, r.Shutdown(ctx), context.Canceled)
		assert.ErrorIs(t, <-done, context.Canceled)
		assert.ErrorIs(t, r.Shutdown(t.Context()), context.Canceled)
	})
}

func TestRouter_RunCancellationPropagatesContextAndDiscardsPending(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		type key struct{}
		parent := context.WithValue(t.Context(), key{}, "value")
		parent, cancel := context.WithTimeout(parent, time.Hour)
		defer cancel()
		r := quietRouter()
		started := make(chan context.Context, 1)
		var calls atomic.Int32
		require.NoError(t, r.Listen(os.Interrupt, func(ctx context.Context, _ os.Signal) error {
			calls.Add(1)
			started <- ctx
			<-ctx.Done()
			return nil
		}))
		source, done := startRouter(t, parent, r)
		source.emit(os.Interrupt)
		handlerCtx := <-started
		assert.Equal(t, "value", handlerCtx.Value(key{}))
		wantDeadline, _ := parent.Deadline()
		gotDeadline, ok := handlerCtx.Deadline()
		assert.True(t, ok)
		assert.Equal(t, wantDeadline, gotDeadline)
		source.emit(os.Interrupt)
		synctest.Wait()
		cancel()
		assert.ErrorIs(t, <-done, context.Canceled)
		assert.Equal(t, int32(1), calls.Load())
		assert.ErrorIs(t, handlerCtx.Err(), context.Canceled)
		assert.ErrorIs(t, r.Shutdown(t.Context()), context.Canceled)
		assert.Equal(t, int32(1), source.stopCalls.Load())
	})
}

func TestRouter_RunCancellationDuringGracefulShutdown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		r := quietRouter()
		started := make(chan struct{})
		require.NoError(t, r.Listen(os.Interrupt, func(ctx context.Context, _ os.Signal) error {
			close(started)
			<-ctx.Done()
			return nil
		}))
		source, done := startRouter(t, ctx, r)
		source.emit(os.Interrupt)
		<-started
		shutdown := make(chan error, 1)
		go func() { shutdown <- r.Shutdown(t.Context()) }()
		synctest.Wait()
		assert.Empty(t, shutdown)
		cancel()
		assert.ErrorIs(t, <-shutdown, context.Canceled)
		assert.ErrorIs(t, <-done, context.Canceled)
	})
}

func TestRouter_EmptyRouterDoesNotSubscribeToAllSignals(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := quietRouter()
		source := newFakeSource()
		r.source = source
		done := make(chan error, 1)
		go func() { done <- r.Run(t.Context()) }()
		synctest.Wait()
		assert.Empty(t, done)
		assert.Equal(t, routerRunning, stateOf(r))
		require.NoError(t, r.Shutdown(t.Context()))
		assert.NoError(t, <-done)
		assert.Zero(t, source.notifyCalls.Load())
		assert.Zero(t, source.stopCalls.Load())
	})
}

func TestRouter_CanceledRunDoesNotSubscribe(t *testing.T) {
	r := quietRouter()
	source := newFakeSource()
	r.source = source
	var calls atomic.Int32
	require.NoError(t, r.Listen(os.Interrupt, func(context.Context, os.Signal) error { calls.Add(1); return nil }))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	assert.ErrorIs(t, r.Run(ctx), context.Canceled)
	assert.Zero(t, calls.Load())
	assert.Zero(t, source.notifyCalls.Load())
	assert.Zero(t, source.stopCalls.Load())
	assert.ErrorIs(t, r.Run(t.Context()), ErrAlreadyStarted)
}

func TestRouter_ShutdownBeforeRunAndRepeatedShutdown(t *testing.T) {
	r := quietRouter()
	source := newFakeSource()
	r.source = source
	require.NoError(t, r.Shutdown(t.Context()))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	assert.NoError(t, r.Shutdown(ctx)) // Already complete, even with canceled ctx.
	assert.ErrorIs(t, r.Run(t.Context()), ErrAlreadyStarted)
	assert.ErrorIs(t, r.Register(&objectHandler{}, os.Interrupt), ErrRegistrationClosed)
	assert.Equal(t, routerStopped, stateOf(r))
	assert.Zero(t, source.notifyCalls.Load())
	assert.Zero(t, source.stopCalls.Load())
}

func TestRouter_RunAndRegistrationAreOneShot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := quietRouter()
		require.NoError(t, r.Register(&objectHandler{}, os.Interrupt))
		_, done := startRouter(t, t.Context(), r)
		assert.ErrorIs(t, r.Run(t.Context()), ErrAlreadyStarted)
		assert.ErrorIs(t, r.Register(&objectHandler{}, os.Interrupt), ErrRegistrationClosed)
		require.NoError(t, r.Shutdown(t.Context()))
		assert.NoError(t, <-done)
		assert.ErrorIs(t, r.Run(t.Context()), ErrAlreadyStarted)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		assert.NoError(t, r.Shutdown(ctx))
	})
}

func TestRouter_ConcurrentSafeLifecycle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := quietRouter()
		require.NoError(t, r.Register(&objectHandler{}, os.Interrupt))
		source := newFakeSource()
		r.source = source
		results := make(chan error, 16)
		for range cap(results) {
			go func() { results <- r.Run(t.Context()) }()
		}
		<-source.ready
		shutdowns := make(chan error, 16)
		for range cap(shutdowns) {
			go func() { shutdowns <- r.Shutdown(t.Context()) }()
		}
		for range cap(shutdowns) {
			assert.NoError(t, <-shutdowns)
		}
		var completed int
		for range cap(results) {
			err := <-results
			if err == nil {
				completed++
			} else {
				assert.ErrorIs(t, err, ErrAlreadyStarted)
			}
		}
		assert.Equal(t, 1, completed)
		assert.Equal(t, int32(1), source.notifyCalls.Load())
		assert.Equal(t, int32(1), source.stopCalls.Load())
	})
}

func TestRouter_ConcurrentSafeRegistrationWhileStarting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := quietRouter()
		require.NoError(t, r.Register(&objectHandler{}, os.Interrupt))
		source := newFakeSource()
		r.source = source
		gate := make(chan struct{})
		registered := make(chan error, 16)
		for range cap(registered) {
			go func() {
				<-gate
				registered <- r.Register(&objectHandler{}, os.Interrupt)
			}()
		}
		done := make(chan error, 1)
		go func() { <-gate; done <- r.Run(t.Context()) }()
		close(gate)
		<-source.ready
		var accepted int
		for range cap(registered) {
			if err := <-registered; err == nil {
				accepted++
			} else {
				assert.ErrorIs(t, err, ErrRegistrationClosed)
			}
		}
		assert.Len(t, r.registrations, accepted+1)
		require.NoError(t, r.Shutdown(t.Context()))
		assert.NoError(t, <-done)
	})
}

func TestRouter_ConcurrentShutdownTimeoutCancelsOtherWaiters(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := quietRouter()
		started := make(chan struct{})
		require.NoError(t, r.Listen(os.Interrupt, func(ctx context.Context, _ os.Signal) error {
			close(started)
			<-ctx.Done()
			return nil
		}))
		source, done := startRouter(t, t.Context(), r)
		source.emit(os.Interrupt)
		<-started
		patient := make(chan error, 1)
		go func() { patient <- r.Shutdown(t.Context()) }()
		synctest.Wait()
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		assert.ErrorIs(t, r.Shutdown(ctx), context.DeadlineExceeded)
		assert.ErrorIs(t, <-patient, context.DeadlineExceeded)
		assert.ErrorIs(t, <-done, context.DeadlineExceeded)
		assert.Equal(t, int32(1), source.stopCalls.Load())
	})
}

func TestNewOptions(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	assert.Same(t, logger, New(WithLogger(logger), nil, WithLogger(nil)).config.logger)
	assert.Same(t, slog.Default(), New(nil).config.logger)
}

type objectHandler struct {
	fn func(context.Context, os.Signal) error
}

func (h *objectHandler) Handle(ctx context.Context, sig os.Signal) error {
	if h.fn != nil {
		return h.fn(ctx, sig)
	}
	return nil
}

type pointerSignal struct{}

func (*pointerSignal) Signal()        {}
func (*pointerSignal) String() string { return "pointer signal" }

type sliceSignal []int

func (sliceSignal) Signal()        {}
func (sliceSignal) String() string { return "slice signal" }

type fakeSource struct {
	ready       chan struct{}
	incoming    chan<- os.Signal
	signals     []os.Signal
	notifyCalls atomic.Int32
	stopCalls   atomic.Int32
}

func newFakeSource() *fakeSource {
	return &fakeSource{ready: make(chan struct{})}
}

func (f *fakeSource) Notify(ch chan<- os.Signal, signals ...os.Signal) {
	f.incoming = ch
	f.signals = slices.Clone(signals)
	f.notifyCalls.Add(1)
	close(f.ready)
}

func (f *fakeSource) Stop(chan<- os.Signal) {
	f.stopCalls.Add(1)
}

func (f *fakeSource) emit(sig os.Signal) {
	f.incoming <- sig
}

func startRouter(t *testing.T, ctx context.Context, r *Router) (*fakeSource, <-chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	source := newFakeSource()
	r.source = source
	done := make(chan error, 1)
	go func() { done <- r.Run(ctx) }()
	<-source.ready
	return source, done
}

func quietRouter() *Router {
	return New(WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))
}

func stateOf(r *Router) routerState {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.state
}

type recordingHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (*recordingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *recordingHandler) Handle(_ context.Context, record slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, record.Clone())
	return nil
}

func (h *recordingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *recordingHandler) WithGroup(string) slog.Handler      { return h }

func (h *recordingHandler) withMessage(message string) []slog.Record {
	h.mu.Lock()
	defer h.mu.Unlock()
	var records []slog.Record
	for _, record := range h.records {
		if record.Message == message {
			records = append(records, record.Clone())
		}
	}
	return records
}

func logAttrs(record slog.Record) map[string]any {
	attrs := make(map[string]any)
	record.Attrs(func(attr slog.Attr) bool { attrs[attr.Key] = attr.Value.Any(); return true })
	return attrs
}
