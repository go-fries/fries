package udp

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer(t *testing.T) {
	addr := newUDPAddr(t)
	done := make(chan []byte, 1)
	recoveryErr := make(chan any, 1)

	server := NewServer(
		addr,
		WithHandler(func(msg *Message) {
			done <- append([]byte(nil), msg.Body...)
		}),
		WithRecoveryHandler(func(_ *Message, err any) {
			recoveryErr <- err
		}),
		WithBufSize(1024),
	)

	startErr := make(chan error, 1)
	go func() {
		startErr <- server.Start(t.Context())
	}()

	t.Cleanup(func() {
		require.NoError(t, server.Stop(t.Context()))

		select {
		case err := <-startErr:
			require.ErrorIs(t, err, net.ErrClosed)
		case <-time.After(time.Second):
			require.FailNow(t, "server did not stop")
		}
	})

	require.Eventually(t, func() bool {
		c, err := net.Dial("udp", addr)
		if err != nil {
			return false
		}
		defer c.Close() //nolint:errcheck

		if _, err = c.Write([]byte("test")); err != nil {
			return false
		}

		select {
		case err := <-recoveryErr:
			require.FailNowf(t, "recovery handler called", "%v", err)
		case buf := <-done:
			return string(buf) == "test"
		default:
			return false
		}
		return false
	}, 3*time.Second, 10*time.Millisecond)
}

func TestServerConfiguresReadChannelSize(t *testing.T) {
	server := NewServer(
		"127.0.0.1:0",
		WithReadChanSize(8),
		WithReadChanSize(0),
	)

	assert.Equal(t, 8, cap(server.readChan))
}

func TestServerStartReturnsListenError(t *testing.T) {
	server := NewServer("invalid-address")

	for range 2 {
		err := server.Start(t.Context())

		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrAlreadyStarted)
	}
}

func TestServerRecoversHandlerPanic(t *testing.T) {
	sentinel := errors.New("handler panic")
	message := &Message{Body: []byte("content")}
	var recoveredMessage *Message
	var recovered any
	server := NewServer(
		"127.0.0.1:0",
		WithHandler(func(*Message) {
			panic(sentinel)
		}),
		WithRecoveryHandler(func(message *Message, err any) {
			recoveredMessage = message
			recovered = err
		}),
	)

	require.NotPanics(t, func() {
		server.handle(message)
	})
	assert.Same(t, message, recoveredMessage)
	assert.Equal(t, sentinel, recovered)
}

func TestServerStopBeforeStart(t *testing.T) {
	server := NewServer("127.0.0.1:0")

	require.NoError(t, server.Stop(t.Context()))
}

func newUDPAddr(t *testing.T) string {
	t.Helper()

	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	defer conn.Close() //nolint:errcheck

	return conn.LocalAddr().String()
}
