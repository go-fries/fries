package locker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLocker_NoopLocker(t *testing.T) {
	l := NoopLocker{}
	ctx := t.Context()

	assert.NoError(t, l.Try(ctx, func() {}))
	assert.NoError(t, l.Until(ctx, time.Second, func() {}))

	o, err := l.Get(ctx)
	assert.NoError(t, err)
	assert.Nil(t, o)

	assert.NoError(t, l.Release(ctx, nil))
	assert.NoError(t, l.ForceRelease(ctx))

	o, err = l.LockedOwner(ctx)
	assert.NoError(t, err)
	assert.Nil(t, o)
}
