package ratelimit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubStore struct {
	takeRequest TakeRequest
	decision    Decision
	takeErr     error
	resetKey    string
	resetErr    error
}

func (s *stubStore) Take(
	_ context.Context,
	request TakeRequest,
) (Decision, error) {
	s.takeRequest = request
	return s.decision, s.takeErr
}

func (s *stubStore) Reset(_ context.Context, key string) error {
	s.resetKey = key
	return s.resetErr
}

func TestNew(t *testing.T) {
	limiter, err := New(&stubStore{}, PerSecond(1))
	require.NoError(t, err)
	assert.NotNil(t, limiter)

	limiter, err = New(nil, PerSecond(1))
	assert.Nil(t, limiter)
	assert.ErrorIs(t, err, ErrNilStore)

	limiter, err = New(&stubStore{}, Limit{})
	assert.Nil(t, limiter)
	assert.ErrorIs(t, err, ErrInvalidLimit)
}

func TestAllowAndAllowN(t *testing.T) {
	limit := Limit{Rate: 2, Period: time.Second, Burst: 4}
	store := &stubStore{
		decision: Decision{
			Allowed:    true,
			Remaining:  2,
			ResetAfter: time.Second,
		},
	}
	limiter, err := New(store, limit)
	require.NoError(t, err)

	decision, err := limiter.Allow(t.Context(), "user:1")
	require.NoError(t, err)
	assert.Equal(t, TakeRequest{Key: "user:1", Limit: limit, Cost: 1}, store.takeRequest)
	assert.Equal(t, limit, decision.Limit)
	assert.True(t, decision.Allowed)

	decision, err = limiter.AllowN(t.Context(), "user:2", 3)
	require.NoError(t, err)
	assert.Equal(t, TakeRequest{Key: "user:2", Limit: limit, Cost: 3}, store.takeRequest)
	assert.Equal(t, limit, decision.Limit)
}

func TestAllowReturnsRejectedDecisionWithoutError(t *testing.T) {
	store := &stubStore{
		decision: Decision{
			Allowed:    false,
			RetryAfter: time.Second,
		},
	}
	limiter, err := New(store, PerMinute(10))
	require.NoError(t, err)

	decision, err := limiter.Allow(t.Context(), "user:1")
	require.NoError(t, err)
	assert.False(t, decision.Allowed)
	assert.Equal(t, time.Second, decision.RetryAfter)
}

func TestAllowValidatesArguments(t *testing.T) {
	limiter, err := New(&stubStore{}, Limit{
		Rate:   1,
		Period: time.Second,
		Burst:  2,
	})
	require.NoError(t, err)

	//nolint:staticcheck // Verifies the documented nil Context behavior.
	_, err = limiter.Allow(nil, "key")
	assert.ErrorIs(t, err, ErrInvalidContext)

	expected := errors.New("canceled")
	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(expected)
	_, err = limiter.Allow(ctx, "key")
	assert.ErrorIs(t, err, expected)

	_, err = limiter.Allow(t.Context(), "")
	assert.ErrorIs(t, err, ErrInvalidKey)

	for _, cost := range []int{-1, 0, 3} {
		_, err = limiter.AllowN(t.Context(), "key", cost)
		assert.ErrorIs(t, err, ErrInvalidCost)
	}
}

func TestAllowWrapsStoreError(t *testing.T) {
	expected := errors.New("store unavailable")
	limiter, err := New(&stubStore{takeErr: expected}, PerSecond(1))
	require.NoError(t, err)

	_, err = limiter.Allow(t.Context(), "user:1")
	assert.ErrorIs(t, err, expected)
	assert.Contains(t, err.Error(), `take "user:1"`)
}

func TestReset(t *testing.T) {
	store := &stubStore{}
	limiter, err := New(store, PerSecond(1))
	require.NoError(t, err)

	require.NoError(t, limiter.Reset(t.Context(), "user:1"))
	assert.Equal(t, "user:1", store.resetKey)

	//nolint:staticcheck // Verifies the documented nil Context behavior.
	assert.ErrorIs(t, limiter.Reset(nil, "key"), ErrInvalidContext)
	assert.ErrorIs(t, limiter.Reset(t.Context(), ""), ErrInvalidKey)

	expected := errors.New("store unavailable")
	store.resetErr = expected
	err = limiter.Reset(t.Context(), "user:1")
	assert.ErrorIs(t, err, expected)
	assert.Contains(t, err.Error(), `reset "user:1"`)
}
