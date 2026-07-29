package idempotency

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testValue struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func TestDoValueExecutesAndReplaysValue(t *testing.T) {
	var stored []byte
	var owner string
	completed := false
	store := unusedStore()
	store.begin = func(_ context.Context, request BeginRequest) (BeginResult, error) {
		if completed {
			return BeginResult{
				Status: BeginCompleted,
				Result: append([]byte(nil), stored...),
			}, nil
		}
		owner = request.Token
		return BeginResult{Status: BeginAcquired}, nil
	}
	store.complete = func(_ context.Context, request CompleteRequest) error {
		assert.Equal(t, owner, request.Token)
		assert.Equal(t, defaultResultTTL, request.TTL)
		stored = append([]byte(nil), request.Result...)
		completed = true
		return nil
	}

	executor := New(store)
	calls := 0
	handler := func(context.Context) (testValue, error) {
		calls++
		return testValue{ID: 1, Name: "order"}, nil
	}

	first, err := DoValue(t.Context(), executor, "key", handler)
	require.NoError(t, err)
	assert.Equal(t, testValue{ID: 1, Name: "order"}, first.Value)
	assert.False(t, first.Replayed)
	assert.JSONEq(t, `{"id":1,"name":"order"}`, string(stored))

	second, err := DoValue(t.Context(), executor, "key", handler)
	require.NoError(t, err)
	assert.Equal(t, first.Value, second.Value)
	assert.True(t, second.Replayed)
	assert.Equal(t, 1, calls)
}

func TestDoValueExecutesAndReplaysPointerValue(t *testing.T) {
	var stored []byte
	completed := false
	store := unusedStore()
	store.begin = func(context.Context, BeginRequest) (BeginResult, error) {
		if completed {
			return BeginResult{
				Status: BeginCompleted,
				Result: append([]byte(nil), stored...),
			}, nil
		}
		return BeginResult{Status: BeginAcquired}, nil
	}
	store.complete = func(_ context.Context, request CompleteRequest) error {
		stored = append([]byte(nil), request.Result...)
		completed = true
		return nil
	}

	invalidDestination := errors.New("invalid decode destination")
	executor := New(store, WithCodec(&testCodec{
		marshal: func(value any) ([]byte, error) {
			typed, ok := value.(*testValue)
			if !ok {
				return nil, errors.New("invalid encode value")
			}
			return json.Marshal(typed)
		},
		unmarshal: func(data []byte, destination any) error {
			typed, ok := destination.(*testValue)
			if !ok {
				return invalidDestination
			}
			return json.Unmarshal(data, typed)
		},
	}))
	handler := func(context.Context) (*testValue, error) {
		return &testValue{ID: 1, Name: "order"}, nil
	}

	first, err := DoValue(t.Context(), executor, "key", handler)
	require.NoError(t, err)
	assert.Equal(t, &testValue{ID: 1, Name: "order"}, first.Value)
	assert.False(t, first.Replayed)

	second, err := DoValue(t.Context(), executor, "key", handler)
	require.NoError(t, err)
	assert.Equal(t, first.Value, second.Value)
	assert.NotSame(t, first.Value, second.Value)
	assert.True(t, second.Replayed)
}

func TestDoValueValidatesArguments(t *testing.T) {
	executor := New(unusedStore())
	handler := func(context.Context) (string, error) {
		return "", nil
	}

	assert.Panics(t, func() {
		_, _ = DoValue(t.Context(), nil, "key", handler)
	})
	//nolint:staticcheck // Verifies the documented nil Context behavior.
	_, err := DoValue(nil, executor, "key", handler)
	assert.ErrorIs(t, err, ErrInvalidContext)
	_, err = DoValue(t.Context(), executor, "", handler)
	assert.ErrorIs(t, err, ErrInvalidKey)
	assert.Panics(t, func() {
		_, _ = DoValue[string](t.Context(), executor, "key", nil)
	})
}

func TestDoValueReturnsInProgress(t *testing.T) {
	store := unusedStore()
	store.begin = func(context.Context, BeginRequest) (BeginResult, error) {
		return BeginResult{Status: BeginInProgress}, nil
	}

	result, err := DoValue(
		t.Context(),
		New(store),
		"key",
		func(context.Context) (string, error) {
			panic("unexpected handler")
		},
	)

	assert.ErrorIs(t, err, ErrInProgress)
	assert.Zero(t, result)
}

func TestDoValueAbortsFailedHandlerAndReturnsValue(t *testing.T) {
	handlerErr := errors.New("handler failed")
	abortErr := errors.New("abort failed")
	store := unusedStore()
	store.begin = func(context.Context, BeginRequest) (BeginResult, error) {
		return BeginResult{Status: BeginAcquired}, nil
	}
	store.abort = func(context.Context, AbortRequest) error {
		return abortErr
	}

	result, err := DoValue(
		t.Context(),
		New(store),
		"key",
		func(context.Context) (string, error) {
			return "partial", handlerErr
		},
	)

	assert.Equal(t, "partial", result.Value)
	assert.False(t, result.Replayed)
	assert.ErrorIs(t, err, handlerErr)
	assert.ErrorIs(t, err, abortErr)
}

func TestDoValueEncodingFailureKeepsClaim(t *testing.T) {
	encodeErr := errors.New("encode failed")
	aborted := false
	completed := false
	store := unusedStore()
	store.begin = func(context.Context, BeginRequest) (BeginResult, error) {
		return BeginResult{Status: BeginAcquired}, nil
	}
	store.abort = func(context.Context, AbortRequest) error {
		aborted = true
		return nil
	}
	store.complete = func(context.Context, CompleteRequest) error {
		completed = true
		return nil
	}
	executor := New(store, WithCodec(&testCodec{
		marshal: func(any) ([]byte, error) {
			return nil, encodeErr
		},
	}))

	result, err := DoValue(
		t.Context(),
		executor,
		"key",
		func(context.Context) (string, error) {
			return "created", nil
		},
	)

	assert.Equal(t, "created", result.Value)
	assert.ErrorIs(t, err, encodeErr)
	assert.False(t, aborted)
	assert.False(t, completed)
}

func TestDoValueReturnsCompleteErrorWithValue(t *testing.T) {
	completeErr := errors.New("complete failed")
	store := unusedStore()
	store.begin = func(context.Context, BeginRequest) (BeginResult, error) {
		return BeginResult{Status: BeginAcquired}, nil
	}
	store.complete = func(context.Context, CompleteRequest) error {
		return completeErr
	}

	result, err := DoValue(
		t.Context(),
		New(store),
		"key",
		func(context.Context) (string, error) {
			return "created", nil
		},
		WithResultTTL(time.Hour),
	)

	assert.Equal(t, "created", result.Value)
	assert.ErrorIs(t, err, completeErr)
}

func TestDoValueReturnsDecodeErrorForReplayedResult(t *testing.T) {
	decodeErr := errors.New("decode failed")
	store := unusedStore()
	store.begin = func(context.Context, BeginRequest) (BeginResult, error) {
		return BeginResult{
			Status: BeginCompleted,
			Result: []byte("stored"),
		}, nil
	}
	executor := New(store, WithCodec(&testCodec{
		unmarshal: func([]byte, any) error {
			return decodeErr
		},
	}))

	result, err := DoValue(
		t.Context(),
		executor,
		"key",
		func(context.Context) (string, error) {
			panic("unexpected handler")
		},
	)

	assert.True(t, result.Replayed)
	assert.ErrorIs(t, err, decodeErr)
}
