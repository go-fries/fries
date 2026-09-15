package crontab

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/flc1125/go-cron/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type lifecycleServer interface {
	Start(context.Context) error
	Stop(context.Context) error
}

type onceSchedule struct {
	next time.Time
	used bool
}

func (s *onceSchedule) Next(time.Time) time.Time {
	if s.used {
		return time.Time{}
	}
	s.used = true
	return s.next
}

func TestNewServer(t *testing.T) {
	t.Parallel()

	c := cron.New()
	server := NewServer(c)

	require.NotNil(t, server)
	assert.Same(t, c, server.Cron())
}

func TestNewServer_WithLogger(t *testing.T) {
	t.Parallel()

	backend := &recordingLogger{}
	logger := slog.New(backend)
	opts := []ServerOption{WithLogger(logger), nil}
	server := NewServer(cron.New(), opts...)

	assert.Same(t, logger, server.logger)
}

func TestServer_ImplementsLifecycleServer(t *testing.T) {
	t.Parallel()

	var _ lifecycleServer = NewServer(cron.New())
}

func TestServer_StartRunsUntilStop(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		server := NewServer(cron.New())
		defer func() {
			assert.NoError(t, server.Stop(context.WithoutCancel(t.Context())))
			synctest.Wait()
		}()
		done := make(chan error, 1)

		go func() {
			done <- server.Start(t.Context())
		}()

		synctest.Wait()
		assert.True(t, server.Cron().IsRunning())
		require.Empty(t, done)
		require.NoError(t, server.Stop(t.Context()))
		synctest.Wait()
		assert.False(t, server.Cron().IsRunning())
		require.Len(t, done, 1)
		require.NoError(t, <-done)
	})
}

func TestServer_StopBeforeStart(t *testing.T) {
	t.Parallel()

	server := NewServer(cron.New())

	require.NoError(t, server.Stop(t.Context()))
}

func TestServer_StopRespectsContextWhileWaitingForJobs(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		started := make(chan struct{}, 1)
		completed := make(chan struct{}, 1)
		release := make(chan struct{})
		unblock := sync.OnceFunc(func() { close(release) })
		c := cron.New()
		c.Schedule(&onceSchedule{next: time.Now().Add(time.Second)}, cron.JobFunc(func(context.Context) error {
			started <- struct{}{}
			<-release
			completed <- struct{}{}
			return nil
		}))
		server := NewServer(c)
		defer func() {
			unblock()
			assert.NoError(t, server.Stop(context.WithoutCancel(t.Context())))
			synctest.Wait()
		}()
		done := make(chan error, 1)

		go func() {
			done <- server.Start(t.Context())
		}()

		synctest.Wait()
		assert.True(t, c.IsRunning())
		time.Sleep(time.Second - time.Nanosecond)
		synctest.Wait()
		require.Empty(t, started)
		time.Sleep(time.Nanosecond)
		synctest.Wait()
		require.Len(t, started, 1)
		require.Empty(t, done)

		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		stopped := make(chan error, 1)
		go func() { stopped <- server.Stop(ctx) }()
		synctest.Wait()
		assert.False(t, c.IsRunning())
		require.Empty(t, stopped)
		time.Sleep(time.Second - time.Nanosecond)
		synctest.Wait()
		require.Empty(t, stopped, "Stop returned before its deadline")
		require.Empty(t, completed)
		time.Sleep(time.Nanosecond)
		synctest.Wait()
		require.Len(t, stopped, 1)
		require.ErrorIs(t, <-stopped, context.DeadlineExceeded)
		require.Empty(t, completed, "deadline must not imply job completion")
		unblock()
		require.NoError(t, server.Stop(t.Context()))
		synctest.Wait()
		require.Len(t, completed, 1)
		require.Len(t, done, 1)
		require.NoError(t, <-done)
	})
}
