package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Deliberately implements only the two methods needed by the typed view.
type typedReadWriter struct {
	get func(context.Context, string, any) error
	put func(context.Context, string, any, time.Duration) (bool, error)
}

func (s typedReadWriter) Get(ctx context.Context, key string, dest any) error {
	return s.get(ctx, key, dest)
}

func (s typedReadWriter) Put(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	return s.put(ctx, key, value, ttl)
}

func TestTypedViewSharesRepository(t *testing.T) {
	t.Parallel()

	type user struct{ Name string }
	repo := NewRepository(newUtilMockStore())
	users := Typed[user](repo)
	ctx := t.Context()

	missing, err := users.Get(ctx, "users:42")
	require.ErrorIs(t, err, ErrNotFound)
	assert.Zero(t, missing)

	want := user{Name: "Alice"}
	loaded, err := users.Remember(ctx, "users:42", time.Minute, func(context.Context) (user, error) {
		return want, nil
	})
	require.NoError(t, err)
	assert.Equal(t, want, loaded)

	// A hit does not need a loader; the same view can be reused.
	cached, err := users.Remember(ctx, "users:42", time.Minute, nil)
	require.NoError(t, err)
	assert.Equal(t, want, cached)

	want.Name = "Bob"
	require.NoError(t, users.Set(ctx, "users:42", want, time.Minute))
	got, err := Get[user](ctx, repo, "users:42")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestTypedViewSetResults(t *testing.T) {
	t.Parallel()

	backendErr := fmt.Errorf("backend: %w", assert.AnError)
	for _, tt := range []struct {
		name    string
		status  bool
		err     error
		wantErr error
	}{
		{name: "confirmed", status: true},
		{name: "unconfirmed", wantErr: ErrWriteUnconfirmed},
		{name: "backend error", err: backendErr, wantErr: backendErr},
		{name: "error takes precedence", status: true, err: backendErr, wantErr: backendErr},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			value := &struct{ Name string }{Name: "Alice"}
			calls := 0
			view := Typed[*struct{ Name string }](typedReadWriter{
				put: func(received context.Context, key string, got any, ttl time.Duration) (bool, error) {
					calls++
					assert.Same(t, ctx, received)
					assert.Equal(t, "users:42", key)
					assert.Same(t, value, got)
					assert.Equal(t, 37*time.Second, ttl)
					return tt.status, tt.err
				},
			})
			err := view.Set(ctx, "users:42", value, 37*time.Second)
			assert.ErrorIs(t, err, tt.wantErr)
			if tt.wantErr != nil {
				assert.Same(t, tt.wantErr, err)
			}
			assert.Equal(t, 1, calls)
		})
	}
}

func TestTypedViewRememberResults(t *testing.T) {
	t.Parallel()

	wrappedMiss := fmt.Errorf("backend: %w", ErrNotFound)
	backendErr := fmt.Errorf("backend: %w", assert.AnError)
	for _, tt := range []struct {
		name      string
		cached    string
		readErr   error
		loadErr   error
		put       bool
		putErr    error
		want      string
		wantErr   error
		wantCalls []string
	}{
		{name: "hit", cached: "cached", want: "cached", wantCalls: []string{"get"}},
		{name: "zero value hit", wantCalls: []string{"get"}},
		{name: "read error with partial value", cached: "partial", readErr: backendErr, want: "partial", wantErr: backendErr, wantCalls: []string{"get"}},
		{name: "miss", readErr: ErrNotFound, put: true, want: "loaded", wantCalls: []string{"get", "load", "put"}},
		{name: "wrapped miss", readErr: wrappedMiss, put: true, want: "loaded", wantCalls: []string{"get", "load", "put"}},
		{name: "loader error with value", readErr: ErrNotFound, loadErr: backendErr, want: "loaded", wantErr: backendErr, wantCalls: []string{"get", "load"}},
		{name: "write error with value", readErr: ErrNotFound, putErr: backendErr, want: "loaded", wantErr: backendErr, wantCalls: []string{"get", "load", "put"}},
		{name: "unconfirmed write with value", readErr: ErrNotFound, want: "loaded", wantErr: ErrWriteUnconfirmed, wantCalls: []string{"get", "load", "put"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			var calls []string
			view := Typed[string](typedReadWriter{
				get: func(received context.Context, key string, dest any) error {
					calls = append(calls, "get")
					assert.Same(t, ctx, received)
					assert.Equal(t, "key", key)
					*dest.(*string) = tt.cached
					return tt.readErr
				},
				put: func(received context.Context, key string, value any, ttl time.Duration) (bool, error) {
					calls = append(calls, "put")
					assert.Same(t, ctx, received)
					assert.Equal(t, "key", key)
					assert.Equal(t, "loaded", value)
					assert.Equal(t, time.Minute, ttl)
					return tt.put, tt.putErr
				},
			})
			value, err := view.Remember(ctx, "key", time.Minute, func(received context.Context) (string, error) {
				calls = append(calls, "load")
				assert.Same(t, ctx, received)
				return "loaded", tt.loadErr
			})
			assert.Equal(t, tt.want, value)
			assert.ErrorIs(t, err, tt.wantErr)
			if tt.wantErr != nil {
				assert.Same(t, tt.wantErr, err)
			}
			assert.Equal(t, tt.wantCalls, calls)
		})
	}
}

func TestTypedViewCodecErrors(t *testing.T) {
	t.Parallel()

	t.Run("decode", func(t *testing.T) {
		type user struct {
			Name string
			Age  int
		}
		view := Typed[user](typedReadWriter{
			get: func(_ context.Context, _ string, dest any) error {
				return json.Unmarshal([]byte(`{"Name":"Alice","Age":"invalid"}`), dest)
			},
		})
		value, err := view.Get(t.Context(), "user")
		var decodeErr *json.UnmarshalTypeError
		require.ErrorAs(t, err, &decodeErr)
		assert.Equal(t, "Alice", value.Name)
	})

	t.Run("encode on write back", func(t *testing.T) {
		view := Typed[chan int](typedReadWriter{
			get: func(context.Context, string, any) error { return ErrNotFound },
			put: func(_ context.Context, _ string, value any, _ time.Duration) (bool, error) {
				_, err := json.Marshal(value)
				return err == nil, err
			},
		})
		loaded := make(chan int)
		value, err := view.Remember(t.Context(), "key", time.Minute, func(context.Context) (chan int, error) {
			return loaded, nil
		})
		var encodeErr *json.UnsupportedTypeError
		require.ErrorAs(t, err, &encodeErr)
		assert.Equal(t, loaded, value)
	})
}

func TestTypedViewCancellation(t *testing.T) {
	t.Parallel()

	t.Run("read cancellation skips loader", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		view := Typed[string](typedReadWriter{
			get: func(ctx context.Context, _ string, _ any) error { return ctx.Err() },
		})
		_, err := view.Remember(ctx, "key", time.Minute, nil)
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("loader cancellation skips write", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		view := Typed[string](typedReadWriter{
			get: func(context.Context, string, any) error { return ErrNotFound },
		})
		_, err := view.Remember(ctx, "key", time.Minute, func(received context.Context) (string, error) {
			cancel()
			return "", received.Err()
		})
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("write cancellation preserves loaded value", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		view := Typed[string](typedReadWriter{
			get: func(context.Context, string, any) error { return ErrNotFound },
			put: func(ctx context.Context, _ string, _ any, _ time.Duration) (bool, error) {
				return false, ctx.Err()
			},
		})
		value, err := view.Remember(ctx, "key", time.Minute, func(context.Context) (string, error) {
			cancel()
			return "loaded", nil
		})
		require.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, "loaded", value)
	})
}
