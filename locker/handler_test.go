package locker

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testLock struct {
	lease        Lease
	acquireErr   error
	tryErr       error
	acquireCalls int
	tryCalls     int
}

func (l *testLock) Acquire(context.Context) (Lease, error) {
	l.acquireCalls++
	return l.lease, l.acquireErr
}

func (l *testLock) TryAcquire(context.Context) (Lease, error) {
	l.tryCalls++
	return l.lease, l.tryErr
}

type testLease struct {
	release func(context.Context) error
}

func (l *testLease) Release(ctx context.Context) error {
	return l.release(ctx)
}

func TestDo(t *testing.T) {
	type contextKey struct{}

	handlerErr := errors.New("handler failed")
	releaseErr := errors.New("release failed")
	key := contextKey{}
	parent, cancel := context.WithCancel(context.WithValue(t.Context(), key, "value"))

	releaseCalled := false
	lock := &testLock{lease: &testLease{release: func(ctx context.Context) error {
		releaseCalled = true
		assert.NoError(t, ctx.Err())
		_, hasDeadline := ctx.Deadline()
		assert.False(t, hasDeadline)
		assert.Equal(t, "value", ctx.Value(key))
		return releaseErr
	}}}

	err := Do(parent, lock, func(ctx context.Context) error {
		cancel()
		assert.ErrorIs(t, ctx.Err(), context.Canceled)
		return handlerErr
	})

	assert.Equal(t, 1, lock.acquireCalls)
	assert.Zero(t, lock.tryCalls)
	assert.True(t, releaseCalled)
	assert.ErrorIs(t, err, handlerErr)
	assert.ErrorIs(t, err, releaseErr)
}

func TestTry(t *testing.T) {
	releaseCalled := false
	lock := &testLock{lease: &testLease{release: func(context.Context) error {
		releaseCalled = true
		return nil
	}}}

	err := Try(t.Context(), lock, func(context.Context) error { return nil })
	require.NoError(t, err)
	assert.Zero(t, lock.acquireCalls)
	assert.Equal(t, 1, lock.tryCalls)
	assert.True(t, releaseCalled)
}

func TestHandlerSingleErrorPreservesIdentity(t *testing.T) {
	handlerErr := errors.New("handler failed")
	releaseErr := errors.New("release failed")
	tests := []struct {
		name       string
		handlerErr error
		releaseErr error
		want       error
	}{
		{
			name:       "handler error",
			handlerErr: handlerErr,
			want:       handlerErr,
		},
		{
			name:       "release error",
			releaseErr: releaseErr,
			want:       releaseErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lock := &testLock{lease: &testLease{release: func(context.Context) error {
				return tt.releaseErr
			}}}

			err := Do(t.Context(), lock, func(context.Context) error {
				return tt.handlerErr
			})

			assert.True(t, err == tt.want)
		})
	}
}

func TestTryNotAcquired(t *testing.T) {
	lock := &testLock{tryErr: ErrNotAcquired}
	handlerCalled := false

	err := Try(t.Context(), lock, func(context.Context) error {
		handlerCalled = true
		return nil
	})

	assert.ErrorIs(t, err, ErrNotAcquired)
	assert.False(t, handlerCalled)
}

func TestHandlerValidation(t *testing.T) {
	validLock := &testLock{}
	validHandler := func(context.Context) error { return nil }

	tests := []struct {
		name    string
		ctx     context.Context
		lock    Lock
		handler Handler
		err     error
	}{
		{name: "nil context", lock: validLock, handler: validHandler, err: ErrInvalidContext},
		{name: "nil lock", ctx: t.Context(), handler: validHandler, err: ErrNilLock},
		{name: "nil handler", ctx: t.Context(), lock: validLock, err: ErrNilHandler},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ErrorIs(t, Do(tt.ctx, tt.lock, tt.handler), tt.err)
			assert.ErrorIs(t, Try(tt.ctx, tt.lock, tt.handler), tt.err)
		})
	}

	assert.Zero(t, validLock.acquireCalls)
	assert.Zero(t, validLock.tryCalls)
}

func TestHandlerPanicReleasesLease(t *testing.T) {
	releaseCalled := false
	lock := &testLock{lease: &testLease{release: func(context.Context) error {
		releaseCalled = true
		return nil
	}}}

	assert.PanicsWithValue(t, "boom", func() {
		_ = Do(t.Context(), lock, func(context.Context) error {
			panic("boom")
		})
	})
	assert.True(t, releaseCalled)
}
