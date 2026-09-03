//go:build !windows && !plan9

package syslog

import (
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDial(t *testing.T) {
	listener, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, listener.Close())
	})

	handler, err := Dial("udp", listener.LocalAddr().String(), WithTag("fries-test"))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, handler.Close())
	})

	require.NoError(t, handler.Handle(t.Context(), slog.NewRecord(timeNow(), slog.LevelInfo, "service started", 0)))
	require.NoError(t, listener.SetReadDeadline(time.Now().Add(5*time.Second)))

	message := make([]byte, 1024)
	n, _, err := listener.ReadFrom(message)
	require.NoError(t, err)
	assert.Contains(t, string(message[:n]), "fries-test")
	assert.Contains(t, string(message[:n]), "service started")
}

func TestDialReturnsConnectionError(t *testing.T) {
	handler, err := Dial("invalid-network", "", WithTag("fries-test"))

	assert.Nil(t, handler)
	require.Error(t, err)
}
