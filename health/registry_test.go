package health_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
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
	registry := health.New(health.WithConcurrency(3))
	expectedErrors := []error{errors.New("first"), nil, errors.New("third")}
	delays := []time.Duration{30 * time.Millisecond, 10 * time.Millisecond, 0}
	names := []string{"first", "second", "third"}

	for i, name := range names {
		registry.Register(name, health.CheckFunc(func(context.Context) error {
			time.Sleep(delays[i])
			return expectedErrors[i]
		}))
	}

	report := registry.Check(t.Context())

	require.Len(t, report.Results, len(names))
	assert.False(t, report.Healthy())
	for i, result := range report.Results {
		assert.Equal(t, names[i], result.Name)
		assert.ErrorIs(t, result.Err, expectedErrors[i])
	}
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
	const (
		checks      = 8
		concurrency = 4
	)

	registry := health.New(health.WithConcurrency(concurrency))
	started := make(chan struct{}, checks)
	release := make(chan struct{})
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

	for range concurrency {
		<-started
	}
	select {
	case <-started:
		t.Fatal("more checks started than the configured concurrency")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)

	report := <-done
	assert.True(t, report.Healthy())
	assert.LessOrEqual(t, maxActive.Load(), int32(concurrency))
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
	registry := health.New(
		health.WithTimeout(20*time.Millisecond),
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

	report := registry.Check(t.Context())

	assert.Equal(t, int32(1), calls.Load())
	require.Len(t, report.Results, 2)
	assert.ErrorIs(t, report.Results[0].Err, context.DeadlineExceeded)
	assert.ErrorIs(t, report.Results[1].Err, context.DeadlineExceeded)
	assert.False(t, report.Healthy())
}

func TestRegistryCheckMarksLateSuccessCanceled(t *testing.T) {
	registry := health.New(
		health.WithTimeout(5*time.Millisecond),
		health.WithConcurrency(1),
	)
	registry.Register("slow", health.CheckFunc(func(context.Context) error {
		time.Sleep(20 * time.Millisecond)
		return nil
	}))

	report := registry.Check(t.Context())

	require.Len(t, report.Results, 1)
	assert.ErrorIs(t, report.Results[0].Err, context.DeadlineExceeded)
}

func TestRegistryCheckUsesSnapshot(t *testing.T) {
	registry := health.New(health.WithConcurrency(1))
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	registry.Register("first", health.CheckFunc(func(context.Context) error {
		started <- struct{}{}
		<-release
		return nil
	}))

	done := make(chan health.Report, 1)
	go func() {
		done <- registry.Check(t.Context())
	}()
	<-started

	registry.Register("second", health.CheckFunc(func(context.Context) error {
		return nil
	}))
	close(release)

	firstReport := <-done
	secondReport := registry.Check(t.Context())

	require.Len(t, firstReport.Results, 1)
	assert.Equal(t, "first", firstReport.Results[0].Name)
	require.Len(t, secondReport.Results, 2)
	assert.Equal(t, "second", secondReport.Results[1].Name)
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
