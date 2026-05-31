package signal

import (
	"context"
	"os"
	"reflect"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_StartWithoutHandlersStops(t *testing.T) {
	srv := NewServer()
	done := make(chan error, 1)

	go func() {
		done <- srv.Start(t.Context())
	}()

	require.NoError(t, srv.Stop(t.Context()))
	assert.NoError(t, receive(t, done))
}

func TestServer_ServeDispatchesAndRecovers(t *testing.T) {
	sig := syscall.SIGUSR1
	type contextKey struct{}
	ctx := context.WithValue(t.Context(), contextKey{}, "recovery context")
	handled := make(chan os.Signal, 1)
	recovered := make(chan any, 1)
	recoveredCtx := make(chan any, 1)

	srv := NewServer(
		WithRecovery(func(ctx context.Context, _ os.Signal, _ Handler, panicValue any) {
			recoveredCtx <- ctx.Value(contextKey{})
			recovered <- panicValue
		}),
	)
	handlers, signals := buildHandlers([]Handler{
		testHandler{
			signals: []os.Signal{sig},
			handle: func(_ context.Context, sig os.Signal) {
				handled <- sig
			},
		},
		testHandler{
			signals: []os.Signal{sig},
			handle: func(context.Context, os.Signal) {
				panic("handler panic")
			},
		},
	})

	assert.Equal(t, []os.Signal{sig}, signals)

	ch := make(chan os.Signal, 1)
	done := make(chan error, 1)
	go func() {
		done <- srv.serve(ctx, ch, handlers)
	}()

	ch <- sig
	assert.Equal(t, sig, receive(t, handled))
	assert.Equal(t, "recovery context", receive(t, recoveredCtx))
	assert.Equal(t, "handler panic", receive(t, recovered))

	require.NoError(t, srv.Stop(t.Context()))
	assert.NoError(t, receive(t, done))
}

func TestServer_ServeDispatchesAsyncHandlers(t *testing.T) {
	sig := syscall.SIGUSR1
	started := make(chan os.Signal, 1)
	release := make(chan struct{})
	srv := NewServer()
	handlers, _ := buildHandlers([]Handler{
		asyncTestHandler{
			signals: []os.Signal{sig},
			handle: func(_ context.Context, sig os.Signal) {
				started <- sig
				<-release
			},
		},
	})

	ch := make(chan os.Signal, 1)
	done := make(chan error, 1)
	go func() {
		done <- srv.serve(t.Context(), ch, handlers)
	}()

	ch <- sig
	assert.Equal(t, sig, receive(t, started))

	require.NoError(t, srv.Stop(t.Context()))
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		close(release)
		t.Fatal("server did not stop while async handler was running")
	}
	close(release)
}

func TestServer_StopIsIdempotent(t *testing.T) {
	srv := NewServer()

	assert.NotPanics(t, func() {
		assert.NoError(t, srv.Stop(t.Context()))
		assert.NoError(t, srv.Stop(t.Context()))
	})
}

func TestNewServer_DefaultRecovery(t *testing.T) {
	assert.NotNil(t, NewServer().recovery)
	assert.Equal(t, reflect.ValueOf(defaultRecovery).Pointer(), reflect.ValueOf(NewServer(WithRecovery(nil)).recovery).Pointer())
}

func TestNewServer_SkipsNilOptions(t *testing.T) {
	assert.NotPanics(t, func() {
		srv := NewServer(nil)
		assert.NotNil(t, srv.recovery)
	})
}

func TestBuildHandlersCallsListenOnce(t *testing.T) {
	var listenCalls atomic.Int32
	handler := countingHandler{
		signals: []os.Signal{syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGUSR1},
		count:   &listenCalls,
	}

	handlers, signals := buildHandlers([]Handler{handler})

	assert.Equal(t, int32(1), listenCalls.Load())
	assert.Equal(t, []os.Signal{syscall.SIGUSR1, syscall.SIGUSR2}, signals)
	assert.Len(t, handlers[syscall.SIGUSR1], 1)
	assert.Len(t, handlers[syscall.SIGUSR2], 1)
}

func TestBuildHandlersSkipsNilHandlers(t *testing.T) {
	var nilHandler *testHandler
	handler := testHandler{signals: []os.Signal{syscall.SIGUSR1}}

	handlers, signals := buildHandlers([]Handler{nil, nilHandler, handler})

	assert.Equal(t, []os.Signal{syscall.SIGUSR1}, signals)
	assert.Equal(t, []Handler{handler}, handlers[syscall.SIGUSR1])
}

func TestServer_StartReturnsContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	assert.ErrorIs(t, NewServer().Start(ctx), context.Canceled)
}

func TestServer_SnapshotHandlers(t *testing.T) {
	srv := NewServer()
	handler := testHandler{signals: []os.Signal{syscall.SIGUSR1}}
	srv.Register(handler)

	handlers := srv.snapshotHandlers()
	require.Len(t, handlers, 1)

	handlers[0] = nil
	assert.Equal(t, []Handler{handler}, srv.snapshotHandlers())
}

func TestServer_ServePassesContextToHandlers(t *testing.T) {
	sig := syscall.SIGUSR1
	type contextKey struct{}
	ctx := context.WithValue(t.Context(), contextKey{}, "context value")
	handled := make(chan any, 1)
	srv := NewServer()
	handlers, _ := buildHandlers([]Handler{
		testHandler{
			signals: []os.Signal{sig},
			handle: func(ctx context.Context, _ os.Signal) {
				handled <- ctx.Value(contextKey{})
			},
		},
	})

	ch := make(chan os.Signal, 1)
	done := make(chan error, 1)
	go func() {
		done <- srv.serve(ctx, ch, handlers)
	}()

	ch <- sig
	assert.Equal(t, "context value", receive(t, handled))

	require.NoError(t, srv.Stop(t.Context()))
	assert.NoError(t, receive(t, done))
}

func TestServer_RegisterSkipsNilHandlers(t *testing.T) {
	var nilHandler *testHandler
	handler := testHandler{signals: []os.Signal{syscall.SIGUSR1}}
	srv := NewServer(WithHandlers(nil, nilHandler, handler))

	srv.Register(nil, nilHandler)

	assert.Equal(t, []Handler{handler}, srv.snapshotHandlers())
}

type testHandler struct {
	signals []os.Signal
	handle  func(context.Context, os.Signal)
}

func (h testHandler) Listen() []os.Signal {
	return h.signals
}

func (h testHandler) Handle(ctx context.Context, sig os.Signal) {
	if h.handle != nil {
		h.handle(ctx, sig)
	}
}

type asyncTestHandler struct {
	AsyncHandler
	signals []os.Signal
	handle  func(context.Context, os.Signal)
}

func (h asyncTestHandler) Listen() []os.Signal {
	return h.signals
}

func (h asyncTestHandler) Handle(ctx context.Context, sig os.Signal) {
	if h.handle != nil {
		h.handle(ctx, sig)
	}
}

type countingHandler struct {
	signals []os.Signal
	count   *atomic.Int32
}

func (h countingHandler) Listen() []os.Signal {
	h.count.Add(1)
	return h.signals
}

func (h countingHandler) Handle(context.Context, os.Signal) {}

func TestServer_StartStopsPromptly(t *testing.T) {
	srv := NewServer(WithHandlers(testHandler{signals: []os.Signal{syscall.SIGUSR1}}))
	done := make(chan error, 1)

	go func() {
		done <- srv.Start(t.Context())
	}()

	require.NoError(t, srv.Stop(t.Context()))

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("server did not stop")
	}
}

func receive[T any](t *testing.T, ch <-chan T) T {
	t.Helper()

	select {
	case value := <-ch:
		return value
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for channel receive")
		var zero T
		return zero
	}
}
