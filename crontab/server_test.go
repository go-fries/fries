package crontab

import (
	"context"
	"testing"
	"time"

	"github.com/flc1125/go-cron/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestServer_ImplementsLifecycleServer(t *testing.T) {
	t.Parallel()

	var _ interface {
		Start(context.Context) error
		Stop(context.Context) error
	} = NewServer(cron.New())
}

func TestServer_NewServerPanicsWithNilCron(t *testing.T) {
	t.Parallel()

	require.PanicsWithValue(t, "crontab: cron is nil", func() {
		NewServer(nil)
	})
}

func TestServer_CronReturnsUnderlyingCron(t *testing.T) {
	t.Parallel()

	c := cron.New()
	srv := NewServer(c)

	assert.Same(t, c, srv.Cron())
}

func TestServer_CronNilReceiver(t *testing.T) {
	t.Parallel()

	var srv *Server

	assert.Nil(t, srv.Cron())
}

func TestServer_StartBlocksUntilStop(t *testing.T) {
	t.Parallel()

	srv := NewServer(cron.New())
	done := make(chan error, 1)

	go func() {
		done <- srv.Start(t.Context())
	}()

	require.Eventually(t, srv.Cron().IsRunning, time.Second, 10*time.Millisecond)
	require.NoError(t, srv.Stop(t.Context()))

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		require.Fail(t, "timeout waiting for server to stop")
	}
}

func TestServer_StopBeforeStart(t *testing.T) {
	t.Parallel()

	srv := NewServer(cron.New())

	require.NoError(t, srv.Stop(t.Context()))
}

func TestServer_StopRespectsContextWhileWaitingForJobs(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})
	release := make(chan struct{})
	c := cron.New()
	c.Schedule(&onceSchedule{next: time.Now().Add(10 * time.Millisecond)}, cron.JobFunc(func(context.Context) error {
		close(started)
		<-release
		return nil
	}))
	srv := NewServer(c)
	done := make(chan error, 1)

	go func() {
		done <- srv.Start(t.Context())
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		require.Fail(t, "timeout waiting for cron job to start")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()

	require.ErrorIs(t, srv.Stop(ctx), context.DeadlineExceeded)

	close(release)
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		require.Fail(t, "timeout waiting for server to stop")
	}
}
