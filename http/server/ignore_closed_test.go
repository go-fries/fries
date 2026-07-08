package server_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	server "github.com/go-fries/fries/http/server/v4"
	"github.com/stretchr/testify/require"
)

func TestIgnoreServerClosedSuppressesHTTPServerClosed(t *testing.T) {
	srv := &fakeServer{startErr: http.ErrServerClosed}

	err := server.IgnoreServerClosed(srv).Start(t.Context())

	require.NoError(t, err)
}

func TestIgnoreServerClosedPreservesOtherStartErrors(t *testing.T) {
	startErr := errors.New("listen failed")
	srv := &fakeServer{startErr: startErr}

	err := server.IgnoreServerClosed(srv).Start(t.Context())

	require.ErrorIs(t, err, startErr)
}

func TestIgnoreServerClosedDelegatesStop(t *testing.T) {
	stopErr := errors.New("stop failed")
	srv := &fakeServer{stopErr: stopErr}

	err := server.IgnoreServerClosed(srv).Stop(t.Context())

	require.ErrorIs(t, err, stopErr)
	require.True(t, srv.stopped)
}

type fakeServer struct {
	startErr error
	stopErr  error
	stopped  bool
}

func (s *fakeServer) Start(context.Context) error {
	return s.startErr
}

func (s *fakeServer) Stop(context.Context) error {
	s.stopped = true
	return s.stopErr
}
