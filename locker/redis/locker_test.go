package redis

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"github.com/go-fires/fires/v3/locker"
)

var ctx = context.Background()

func newRedis(t *testing.T) redis.UniversalClient {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	t.Cleanup(func() {
		_ = rdb.Close()
	})
	return rdb
}

func TestLocker_Try(t *testing.T) {
	l := NewLocker(newRedis(t),
		WithName("kratos:locker:try"),
		WithTTL(time.Second*100),
	)

	errs := make(chan error, 2)
	ch := make(chan struct{}, 2)
	wg := sync.WaitGroup{}

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- l.Try(ctx, func() {
				ch <- struct{}{}
				time.Sleep(time.Millisecond * 10)
			})
		}()
	}

	wg.Wait()
	close(errs)
	close(ch)

	errs1 := <-errs
	errs2 := <-errs
	assert.True(t, errs1 == nil || errs2 == nil)
	assert.False(t, errs1 == nil && errs2 == nil)

	assert.Len(t, ch, 1)
}

func TestLocker_Until(t *testing.T) {
	l := NewLocker(newRedis(t),
		WithName("kratos:locker:until"),
	)

	ch := make(chan struct{}, 2)
	wg := sync.WaitGroup{}
	start := time.Now()

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.NoError(t, l.Until(ctx, time.Second, func() {
				ch <- struct{}{}
				time.Sleep(time.Millisecond * 10)
			}))
		}()
	}

	wg.Wait()
	close(ch)

	assert.Len(t, ch, 2)
	assert.True(t, time.Since(start) > time.Millisecond*10)
}

func TestLocker_Until_Timeout(t *testing.T) {
	l := NewLocker(newRedis(t),
		WithName("kratos:locker:until:timeout"),
	)

	errs := make(chan error, 2)
	ch := make(chan struct{}, 2)
	wg := sync.WaitGroup{}

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- l.Until(ctx, time.Millisecond*100, func() {
				ch <- struct{}{}
				time.Sleep(time.Millisecond * 200)
			})
		}()
	}

	wg.Wait()
	close(errs)
	close(ch)

	err1 := <-errs
	err2 := <-errs
	assert.True(t, err1 == nil || err2 == nil)
	assert.False(t, err1 == nil && err2 == nil)
	assert.True(t, errors.Is(err1, locker.ErrTimeout) || errors.Is(err2, locker.ErrTimeout))
	assert.False(t, errors.Is(err1, locker.ErrTimeout) && errors.Is(err2, locker.ErrTimeout))

	assert.Len(t, ch, 1)
}

func TestLocker_GetAndReleaseAndLockedOwner(t *testing.T) {
	l := NewLocker(newRedis(t),
		WithName("kratos:locker:release"),
	)

	owner1, err1 := l.Get(ctx)
	assert.NoError(t, err1)
	assert.NotNil(t, owner1)
	owner, err := l.LockedOwner(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, owner)
	assert.Equal(t, owner1.Name(), owner.Name())

	owner2, err2 := l.Get(ctx)
	assert.ErrorIs(t, err2, locker.ErrLocked)
	assert.Nil(t, owner2)

	assert.NoError(t, l.Release(ctx, owner1))

	owner3, err3 := l.Get(ctx)
	assert.NoError(t, err3)
	assert.NotNil(t, owner3)
	assert.NoError(t, owner3.Release(ctx))
}

func TestLocker_ForceRelease(t *testing.T) {
	l := NewLocker(newRedis(t),
		WithName("kratos:locker:force-release"),
	)

	owner1, err1 := l.Get(ctx)
	assert.NoError(t, err1)
	assert.NotNil(t, owner1)

	assert.NoError(t, l.ForceRelease(ctx))

	owner2, err2 := l.Get(ctx)
	assert.NoError(t, err2)
	assert.NotNil(t, owner2)
	assert.NoError(t, owner2.Release(ctx))
}

// TestLocker_Multi tests the locker with multiple goroutines.
// see: https://github.com/go-kratos-ecosystem/components/issues/326
func TestLocker_Multi(t *testing.T) {
	l := NewLocker(newRedis(t),
		WithName("kratos:locker:multi"),
		WithTTL(time.Millisecond*100),
	)

	ch := make(chan struct{}, 2)
	wg := sync.WaitGroup{}
	ii := int64(0)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if atomic.AddInt64(&ii, 1) == 2 {
				time.Sleep(time.Millisecond * 105)
			}
			assert.NoError(t, l.Try(ctx, func() {
				ch <- struct{}{}
				time.Sleep(time.Millisecond * 200)
			}))
		}()
	}

	wg.Wait()
	close(ch)

	assert.Len(t, ch, 2)
}
