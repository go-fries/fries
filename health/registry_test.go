package health_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-fries/fries/health/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pointerChecker struct{}

func (*pointerChecker) Check(context.Context) error {
	return nil
}

func TestRegistryEmpty(t *testing.T) {
	report := health.New().Check(t.Context())

	assert.True(t, report.Healthy())
	assert.NotZero(t, report.StartedAt)
	assert.NotNil(t, report.Results)
	assert.Empty(t, report.Results)
	assert.GreaterOrEqual(t, report.Duration, time.Duration(0))
}

func TestRegistryCheckPreservesRegistrationOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		registry := health.New(health.WithConcurrency(3))
		expectedErrors := []error{errors.New("first"), nil, errors.New("third")}
		delays := []time.Duration{3 * time.Second, time.Second, 0}
		names := []string{"first", "second", "third"}
		completed := make(chan string, len(names))

		for i, name := range names {
			registry.Register(name, health.CheckFunc(func(context.Context) error {
				time.Sleep(delays[i])
				completed <- name
				return expectedErrors[i]
			}))
		}

		started := time.Now()
		report := registry.Check(t.Context())

		require.Len(t, completed, len(names))
		assert.Equal(t, "third", <-completed)
		assert.Equal(t, "second", <-completed)
		assert.Equal(t, "first", <-completed)
		require.Len(t, report.Results, len(names))
		assert.False(t, report.Healthy())
		assert.Equal(t, started, report.StartedAt)
		assert.Equal(t, 3*time.Second, report.Duration)
		for i, result := range report.Results {
			assert.Equal(t, names[i], result.Name)
			assert.ErrorIs(t, result.Err, expectedErrors[i])
			assert.Equal(t, delays[i], result.Duration)
		}
	})
}

func TestRegistryCheckContinuesAfterErrors(t *testing.T) {
	registry := health.New(health.WithConcurrency(1))
	var calls atomic.Int32

	for _, name := range []string{"first", "second", "third"} {
		registry.Register(name, health.CheckFunc(func(context.Context) error {
			calls.Add(1)
			return assert.AnError
		}))
	}

	report := registry.Check(t.Context())

	assert.Equal(t, int32(3), calls.Load())
	assert.Len(t, report.Results, 3)
	for _, result := range report.Results {
		assert.ErrorIs(t, result.Err, assert.AnError)
	}
}

func TestRegistryCheckBoundsConcurrency(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const (
			checks      = 8
			concurrency = 4
		)

		registry := health.New(health.WithConcurrency(concurrency))
		started := make(chan struct{}, checks)
		release := make(chan struct{})
		unblock := sync.OnceFunc(func() { close(release) })
		defer func() {
			unblock()
			synctest.Wait()
		}()
		var (
			active    atomic.Int32
			maxActive atomic.Int32
		)

		for i := range checks {
			registry.Register(string(rune('a'+i)), health.CheckFunc(func(context.Context) error {
				current := active.Add(1)
				for {
					maximum := maxActive.Load()
					if current <= maximum || maxActive.CompareAndSwap(maximum, current) {
						break
					}
				}
				started <- struct{}{}
				<-release
				active.Add(-1)
				return nil
			}))
		}

		done := make(chan health.Report, 1)
		go func() {
			done <- registry.Check(t.Context())
		}()

		synctest.Wait()
		assert.Len(t, started, concurrency)
		assert.Equal(t, int32(concurrency), active.Load())
		require.Empty(t, done, "check returned before blocked checkers were released")
		unblock()
		synctest.Wait()

		require.Len(t, done, 1)
		report := <-done
		assert.True(t, report.Healthy())
		assert.Len(t, report.Results, checks)
		assert.Len(t, started, checks)
		assert.Equal(t, int32(concurrency), maxActive.Load())
		assert.Zero(t, active.Load())
		assert.Zero(t, report.Duration)
	})
}

func TestRegistryCheckCanceledBeforeStart(t *testing.T) {
	registry := health.New()
	var calls atomic.Int32
	for _, name := range []string{"first", "second"} {
		registry.Register(name, health.CheckFunc(func(context.Context) error {
			calls.Add(1)
			return nil
		}))
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	report := registry.Check(ctx)

	assert.Equal(t, int32(0), calls.Load())
	require.Len(t, report.Results, 2)
	for _, result := range report.Results {
		assert.ErrorIs(t, result.Err, context.Canceled)
	}
}

func TestRegistryCheckTimeoutStopsUnstartedChecks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const timeout = time.Second
		registry := health.New(
			health.WithTimeout(timeout),
			health.WithConcurrency(1),
		)
		var calls atomic.Int32
		registry.Register("running", health.CheckFunc(func(ctx context.Context) error {
			calls.Add(1)
			<-ctx.Done()
			return nil
		}))
		registry.Register("waiting", health.CheckFunc(func(context.Context) error {
			calls.Add(1)
			return nil
		}))

		ctx, cancel := context.WithCancel(t.Context())
		defer func() {
			cancel()
			synctest.Wait()
		}()
		started := time.Now()
		done := make(chan health.Report, 1)
		go func() {
			done <- registry.Check(ctx)
		}()

		synctest.Wait()
		assert.Equal(t, int32(1), calls.Load())
		require.Empty(t, done)
		time.Sleep(timeout - time.Nanosecond)
		synctest.Wait()
		assert.Equal(t, int32(1), calls.Load())
		require.Empty(t, done, "check returned before its deadline")
		time.Sleep(time.Nanosecond)
		synctest.Wait()
		require.Len(t, done, 1, "check did not finish at its deadline")
		report := <-done

		assert.Equal(t, int32(1), calls.Load())
		require.Len(t, report.Results, 2)
		assert.Equal(t, "running", report.Results[0].Name)
		assert.Equal(t, "waiting", report.Results[1].Name)
		assert.ErrorIs(t, report.Results[0].Err, context.DeadlineExceeded)
		assert.ErrorIs(t, report.Results[1].Err, context.DeadlineExceeded)
		assert.Equal(t, timeout, report.Results[0].Duration)
		assert.Zero(t, report.Results[1].Duration)
		assert.Equal(t, timeout, report.Duration)
		assert.Equal(t, timeout, time.Since(started))
		assert.False(t, report.Healthy())
	})
}

func TestRegistryCheckMarksLateSuccessCanceled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const (
			timeout       = time.Second
			checkDuration = 2 * time.Second
		)
		registry := health.New(
			health.WithTimeout(timeout),
			health.WithConcurrency(1),
		)
		registry.Register("slow", health.CheckFunc(func(context.Context) error {
			time.Sleep(checkDuration)
			return nil
		}))

		started := time.Now()
		report := registry.Check(t.Context())

		require.Len(t, report.Results, 1)
		assert.ErrorIs(t, report.Results[0].Err, context.DeadlineExceeded)
		assert.Equal(t, checkDuration, report.Results[0].Duration)
		assert.Equal(t, checkDuration, report.Duration)
		assert.Equal(t, checkDuration, time.Since(started))
		assert.False(t, report.Healthy())
	})
}

func TestRegistryCheckJoinsCheckerErrorAndContextCause(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const timeout = time.Second
		registry := health.New(
			health.WithTimeout(timeout),
			health.WithConcurrency(1),
		)
		checkErr := errors.New("check failed")
		registry.Register("slow", health.CheckFunc(func(ctx context.Context) error {
			<-ctx.Done()
			return checkErr
		}))

		report := registry.Check(t.Context())

		require.Len(t, report.Results, 1)
		assert.ErrorIs(t, report.Results[0].Err, checkErr)
		assert.ErrorIs(t, report.Results[0].Err, context.DeadlineExceeded)
		assert.Equal(t, timeout, report.Results[0].Duration)
		assert.Equal(t, timeout, report.Duration)
		assert.False(t, report.Healthy())
	})
}

func TestRegistryCheckRecoversCheckerPanic(t *testing.T) {
	registry := health.New(health.WithConcurrency(1))
	panicErr := errors.New("panic")
	registry.Register("panic", health.CheckFunc(func(context.Context) error {
		panic(panicErr)
	}))
	registry.Register("healthy", health.CheckFunc(func(context.Context) error {
		return nil
	}))

	report := registry.Check(t.Context())

	require.Len(t, report.Results, 2)
	var recovered *health.PanicError
	require.ErrorAs(t, report.Results[0].Err, &recovered)
	assert.Same(t, panicErr, recovered.Value)
	assert.NotEmpty(t, recovered.Stack)
	assert.ErrorIs(t, report.Results[0].Err, panicErr)
	assert.NoError(t, report.Results[1].Err)
}

func TestRegistryCheckUsesSnapshot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		registry := health.New(health.WithConcurrency(1))
		var firstCalls, secondCalls atomic.Int32
		release := make(chan struct{})
		unblock := sync.OnceFunc(func() { close(release) })
		defer func() {
			unblock()
			synctest.Wait()
		}()
		registry.Register("first", health.CheckFunc(func(context.Context) error {
			firstCalls.Add(1)
			<-release
			return nil
		}))

		done := make(chan health.Report, 1)
		go func() {
			done <- registry.Check(t.Context())
		}()
		synctest.Wait()
		assert.Equal(t, int32(1), firstCalls.Load())
		require.Empty(t, done, "check returned before its snapshot was released")

		registry.Register("second", health.CheckFunc(func(context.Context) error {
			secondCalls.Add(1)
			return nil
		}))
		unblock()
		synctest.Wait()

		require.Len(t, done, 1)
		firstReport := <-done
		assert.Zero(t, secondCalls.Load())
		assert.True(t, firstReport.Healthy())
		require.Len(t, firstReport.Results, 1)
		assert.Equal(t, "first", firstReport.Results[0].Name)

		secondReport := registry.Check(t.Context())

		assert.True(t, secondReport.Healthy())
		assert.Equal(t, int32(2), firstCalls.Load())
		assert.Equal(t, int32(1), secondCalls.Load())
		require.Len(t, secondReport.Results, 2)
		assert.Equal(t, "first", secondReport.Results[0].Name)
		assert.Equal(t, "second", secondReport.Results[1].Name)
	})
}

func TestRegistryRegisterPanicsForInvalidArguments(t *testing.T) {
	var (
		nilRegistry *health.Registry
		nilChecker  *pointerChecker
	)

	assert.Panics(t, func() {
		nilRegistry.Register("check", health.CheckFunc(func(context.Context) error {
			return nil
		}))
	})
	assert.Panics(t, func() {
		health.New().Register("", health.CheckFunc(func(context.Context) error {
			return nil
		}))
	})
	assert.Panics(t, func() {
		health.New().Register("check", nil)
	})
	assert.Panics(t, func() {
		health.New().Register("check", nilChecker)
	})

	registry := health.New()
	registry.Register("check", health.CheckFunc(func(context.Context) error {
		return nil
	}))
	assert.Panics(t, func() {
		registry.Register("check", health.CheckFunc(func(context.Context) error {
			return nil
		}))
	})
}

func TestRegistryCheckPanicsForNilArguments(t *testing.T) {
	var registry *health.Registry

	assert.Panics(t, func() {
		registry.Check(t.Context())
	})
	assert.Panics(t, func() {
		health.New().Check(nil) //nolint:staticcheck // Verifies the nil context contract.
	})
}
