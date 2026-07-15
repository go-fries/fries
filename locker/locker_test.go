package locker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoopLocker(t *testing.T) {
	lock := NoopLocker{}.Lock("test", time.Second)

	lease, err := lock.TryAcquire(t.Context())
	require.NoError(t, err)
	require.NotNil(t, lease)
	assert.NoError(t, lease.Release(t.Context()))

	lease, err = lock.Acquire(t.Context())
	require.NoError(t, err)
	require.NotNil(t, lease)

	_, transferable := lease.(TransferableLease)
	assert.False(t, transferable)
	_, renewable := lease.(RenewableLease)
	assert.False(t, renewable)
	_, restorable := lock.(RestorableLock)
	assert.False(t, restorable)
}

func TestNoopLockValidation(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		lock Lock
		err  error
	}{
		{
			name: "nil context",
			lock: NoopLocker{}.Lock("test", time.Second),
			err:  ErrInvalidContext,
		},
		{
			name: "empty name",
			ctx:  t.Context(),
			lock: NoopLocker{}.Lock("", time.Second),
			err:  ErrInvalidName,
		},
		{
			name: "zero ttl",
			ctx:  t.Context(),
			lock: NoopLocker{}.Lock("test", 0),
			err:  ErrInvalidTTL,
		},
		{
			name: "negative ttl",
			ctx:  t.Context(),
			lock: NoopLocker{}.Lock("test", -time.Second),
			err:  ErrInvalidTTL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lease, err := tt.lock.TryAcquire(tt.ctx)
			assert.Nil(t, lease)
			assert.ErrorIs(t, err, tt.err)

			lease, err = tt.lock.Acquire(tt.ctx)
			assert.Nil(t, lease)
			assert.ErrorIs(t, err, tt.err)
		})
	}
}

func TestNoopLeaseReleaseNilContext(t *testing.T) {
	var ctx context.Context
	assert.ErrorIs(t, NoopLease{}.Release(ctx), ErrInvalidContext)
}

func TestNoopLockCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	lock := NoopLocker{}.Lock("test", time.Second)

	lease, err := lock.TryAcquire(ctx)
	assert.Nil(t, lease)
	assert.ErrorIs(t, err, context.Canceled)

	lease, err = lock.Acquire(ctx)
	assert.Nil(t, lease)
	assert.ErrorIs(t, err, context.Canceled)
	assert.ErrorIs(t, NoopLease{}.Release(ctx), context.Canceled)
}
