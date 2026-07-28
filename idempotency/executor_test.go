package idempotency

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testStore struct {
	begin    func(context.Context, BeginRequest) (BeginResult, error)
	complete func(context.Context, CompleteRequest) error
	abort    func(context.Context, AbortRequest) error
}

func (s *testStore) Begin(ctx context.Context, request BeginRequest) (BeginResult, error) {
	return s.begin(ctx, request)
}

func (s *testStore) Complete(ctx context.Context, request CompleteRequest) error {
	return s.complete(ctx, request)
}

func (s *testStore) Abort(ctx context.Context, request AbortRequest) error {
	return s.abort(ctx, request)
}

func TestNewPanicsWithNilStore(t *testing.T) {
	assert.Panics(t, func() {
		New(nil)
	})

	var store *testStore
	assert.Panics(t, func() {
		New(store)
	})
}

func TestDoValidatesArguments(t *testing.T) {
	executor := New(unusedStore())

	//nolint:staticcheck // Verifies the documented nil Context behavior.
	assert.ErrorIs(t, executor.Do(nil, "key", func(context.Context) error {
		return nil
	}), ErrInvalidContext)
	assert.ErrorIs(t, executor.Do(t.Context(), "", func(context.Context) error {
		return nil
	}), ErrInvalidKey)
	assert.Panics(t, func() {
		_ = executor.Do(t.Context(), "key", nil)
	})

	ctx, cancel := context.WithCancelCause(t.Context())
	expected := errors.New("canceled")
	cancel(expected)
	assert.ErrorIs(t, executor.Do(ctx, "key", func(context.Context) error {
		return nil
	}), expected)
}

func TestDoHandlesExistingStates(t *testing.T) {
	tests := []struct {
		name   string
		status BeginStatus
		err    error
	}{
		{name: "in progress", status: BeginInProgress, err: ErrInProgress},
		{name: "completed", status: BeginCompleted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			store := unusedStore()
			store.begin = func(context.Context, BeginRequest) (BeginResult, error) {
				return BeginResult{Status: tt.status}, nil
			}
			err := New(store).Do(t.Context(), "key", func(context.Context) error {
				called = true
				return nil
			})

			assert.ErrorIs(t, err, tt.err)
			assert.False(t, called)
		})
	}
}

func TestDoCompletesAcquiredClaim(t *testing.T) {
	var beginRequest BeginRequest
	var completeRequest CompleteRequest
	store := unusedStore()
	store.begin = func(_ context.Context, request BeginRequest) (BeginResult, error) {
		beginRequest = request
		return BeginResult{Status: BeginAcquired}, nil
	}
	store.complete = func(_ context.Context, request CompleteRequest) error {
		completeRequest = request
		return nil
	}

	called := false
	err := New(
		store,
		WithDefaultExecutionTTL(time.Minute),
		WithDefaultResultTTL(time.Hour),
	).Do(
		t.Context(),
		"orders:1",
		func(context.Context) error {
			called = true
			return nil
		},
		WithExecutionTTL(2*time.Minute),
		WithResultTTL(2*time.Hour),
		WithFingerprint("request-1"),
	)

	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, "orders:1", beginRequest.Key)
	assert.NotEmpty(t, beginRequest.Token)
	assert.Equal(t, "request-1", beginRequest.Fingerprint)
	assert.Equal(t, 2*time.Minute, beginRequest.TTL)
	assert.Equal(t, beginRequest.Token, completeRequest.Token)
	assert.Equal(t, "orders:1", completeRequest.Key)
	assert.Equal(t, 2*time.Hour, completeRequest.TTL)
}

func TestDoAbortsFailedHandler(t *testing.T) {
	handlerErr := errors.New("handler failed")
	abortErr := errors.New("abort failed")
	var abortRequest AbortRequest
	store := unusedStore()
	store.begin = func(context.Context, BeginRequest) (BeginResult, error) {
		return BeginResult{Status: BeginAcquired}, nil
	}
	store.abort = func(_ context.Context, request AbortRequest) error {
		abortRequest = request
		return abortErr
	}

	err := New(store).Do(t.Context(), "key", func(context.Context) error {
		return handlerErr
	})

	assert.ErrorIs(t, err, handlerErr)
	assert.ErrorIs(t, err, abortErr)
	assert.Equal(t, "key", abortRequest.Key)
	assert.NotEmpty(t, abortRequest.Token)
}

func TestDoReturnsCompleteError(t *testing.T) {
	expected := errors.New("complete failed")
	store := unusedStore()
	store.begin = func(context.Context, BeginRequest) (BeginResult, error) {
		return BeginResult{Status: BeginAcquired}, nil
	}
	store.complete = func(context.Context, CompleteRequest) error {
		return expected
	}

	err := New(store).Do(t.Context(), "key", func(context.Context) error {
		return nil
	})

	assert.ErrorIs(t, err, expected)
}

func TestDoFinalizationContextPreservesValuesAndIgnoresCancellation(t *testing.T) {
	type contextKey struct{}
	ctx, cancel := context.WithCancel(context.WithValue(
		t.Context(),
		contextKey{},
		"value",
	))
	store := unusedStore()
	store.begin = func(context.Context, BeginRequest) (BeginResult, error) {
		return BeginResult{Status: BeginAcquired}, nil
	}
	store.complete = func(finalizationCtx context.Context, _ CompleteRequest) error {
		assert.NoError(t, finalizationCtx.Err())
		assert.Equal(t, "value", finalizationCtx.Value(contextKey{}))
		_, hasDeadline := finalizationCtx.Deadline()
		assert.True(t, hasDeadline)
		return nil
	}

	err := New(store).Do(ctx, "key", func(context.Context) error {
		cancel()
		return nil
	})

	require.NoError(t, err)
}

func TestDoAbortsClaimWhenContextIsCanceledDuringBegin(t *testing.T) {
	expected := errors.New("canceled during begin")
	ctx, cancel := context.WithCancelCause(t.Context())
	handlerCalled := false
	abortCalled := false
	store := unusedStore()
	store.begin = func(context.Context, BeginRequest) (BeginResult, error) {
		cancel(expected)
		return BeginResult{Status: BeginAcquired}, nil
	}
	store.abort = func(finalizationCtx context.Context, _ AbortRequest) error {
		abortCalled = true
		assert.NoError(t, finalizationCtx.Err())
		return nil
	}

	err := New(store).Do(ctx, "key", func(context.Context) error {
		handlerCalled = true
		return nil
	})

	assert.ErrorIs(t, err, expected)
	assert.True(t, abortCalled)
	assert.False(t, handlerCalled)
}

func TestDoDoesNotRecoverHandlerPanic(t *testing.T) {
	aborted := false
	store := unusedStore()
	store.begin = func(context.Context, BeginRequest) (BeginResult, error) {
		return BeginResult{Status: BeginAcquired}, nil
	}
	store.abort = func(context.Context, AbortRequest) error {
		aborted = true
		return nil
	}

	assert.PanicsWithValue(t, "boom", func() {
		_ = New(store).Do(t.Context(), "key", func(context.Context) error {
			panic("boom")
		})
	})
	assert.False(t, aborted)
}

func TestDoReturnsStoreAndStateErrors(t *testing.T) {
	beginErr := errors.New("begin failed")
	store := unusedStore()
	store.begin = func(context.Context, BeginRequest) (BeginResult, error) {
		return BeginResult{}, beginErr
	}
	assert.ErrorIs(t, New(store).Do(
		t.Context(),
		"key",
		func(context.Context) error { return nil },
	), beginErr)

	store.begin = func(context.Context, BeginRequest) (BeginResult, error) {
		return BeginResult{Status: 99}, nil
	}
	assert.EqualError(t, New(store).Do(
		t.Context(),
		"key",
		func(context.Context) error { return nil },
	), "idempotency: invalid begin status 99")
}

func unusedStore() *testStore {
	return &testStore{
		begin: func(context.Context, BeginRequest) (BeginResult, error) {
			panic("unexpected Begin")
		},
		complete: func(context.Context, CompleteRequest) error {
			panic("unexpected Complete")
		},
		abort: func(context.Context, AbortRequest) error {
			panic("unexpected Abort")
		},
	}
}
