package udp

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMessageCopiesBody(t *testing.T) {
	address := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12190}
	body := []byte("first-packet")

	message := newMessage(nil, address, body)
	copy(body, "other-packet")

	assert.Same(t, address, message.Addr)
	assert.Equal(t, "first-packet", string(message.Body))
}

func TestServerStartAfterStopReturnsClosedError(t *testing.T) {
	server := NewServer("127.0.0.1:0")
	require.NoError(t, server.Stop(t.Context()))

	err := server.Start(t.Context())

	assert.ErrorIs(t, err, net.ErrClosed)
}

func TestServerStopIsIdempotent(t *testing.T) {
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	server := NewServer("127.0.0.1:0")
	server.conn = conn

	require.NoError(t, server.Stop(t.Context()))
	require.NoError(t, server.Stop(t.Context()))
}

func TestServerEnqueueReturnsClosedErrorAfterStop(t *testing.T) {
	server := NewServer("127.0.0.1:0", WithReadChanSize(1))
	server.readChan <- &Message{}
	require.NoError(t, server.Stop(t.Context()))

	err := server.enqueue(&Message{})

	assert.ErrorIs(t, err, net.ErrClosed)
}

func TestServerServeStopsWhenReadQueueIsFull(t *testing.T) {
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = conn.Close()
	})
	client, err := net.Dial("udp", conn.LocalAddr().String())
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, client.Close())
	})

	server := NewServer("127.0.0.1:0", WithReadChanSize(1))
	server.readChan <- &Message{}
	server.stop()
	_, err = client.Write([]byte("content"))
	require.NoError(t, err)

	err = server.serve(conn)

	assert.ErrorIs(t, err, net.ErrClosed)
}

func TestServerClearsConnectionWhenServeReturns(t *testing.T) {
	server := NewServer(newUDPAddr(t))
	t.Cleanup(func() {
		require.NoError(t, server.Stop(t.Context()))
	})
	startErr := make(chan error, 1)
	go func() {
		startErr <- server.Start(t.Context())
	}()

	var conn net.PacketConn
	require.Eventually(t, func() bool {
		server.connMu.Lock()
		defer server.connMu.Unlock()
		conn = server.conn
		return conn != nil
	}, time.Second, time.Millisecond)
	require.NoError(t, conn.Close())

	select {
	case err := <-startErr:
		assert.ErrorIs(t, err, net.ErrClosed)
	case <-time.After(time.Second):
		require.FailNow(t, "server did not stop")
	}
	server.connMu.Lock()
	defer server.connMu.Unlock()
	assert.Nil(t, server.conn)
}
