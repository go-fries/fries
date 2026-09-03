package udp

import (
	"net"
	"sync"
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
	server := NewServer("invalid-address")
	require.NoError(t, server.Stop(t.Context()))

	err := server.Start(t.Context())

	assert.ErrorIs(t, err, net.ErrClosed)
}

func TestServerRejectsConcurrentStart(t *testing.T) {
	server := NewServer("127.0.0.1:0")
	firstResult := make(chan error, 1)
	go func() {
		firstResult <- server.Start(t.Context())
	}()

	firstConn := waitForServerConnection(t, server)
	t.Cleanup(func() {
		_ = server.Stop(t.Context())
		_ = firstConn.Close()
	})
	secondResult := make(chan error, 1)
	go func() {
		secondResult <- server.Start(t.Context())
	}()

	select {
	case err := <-secondResult:
		assert.ErrorIs(t, err, ErrAlreadyStarted)
	case <-time.After(time.Second):
		require.FailNow(t, "concurrent start did not return")
	}
	require.NoError(t, server.Stop(t.Context()))
	select {
	case err := <-firstResult:
		assert.ErrorIs(t, err, net.ErrClosed)
	case <-time.After(time.Second):
		require.FailNow(t, "first start did not stop")
	}
}

func TestServerStopWhileStarting(t *testing.T) {
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	listenStarted := make(chan struct{})
	releaseListen := make(chan struct{})
	var releaseOnce sync.Once
	server := NewServer("127.0.0.1:0")
	server.listenPacket = func(string, string) (net.PacketConn, error) {
		close(listenStarted)
		<-releaseListen
		return conn, nil
	}
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(releaseListen) })
		_ = server.Stop(t.Context())
		_ = conn.Close()
	})
	startResult := make(chan error, 1)
	go func() {
		startResult <- server.Start(t.Context())
	}()
	<-listenStarted

	require.NoError(t, server.Stop(t.Context()))
	releaseOnce.Do(func() { close(releaseListen) })

	select {
	case err := <-startResult:
		assert.ErrorIs(t, err, net.ErrClosed)
	case <-time.After(time.Second):
		require.FailNow(t, "start did not stop")
	}
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

	conn := waitForServerConnection(t, server)
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

func waitForServerConnection(t *testing.T, server *Server) net.PacketConn {
	t.Helper()

	var conn net.PacketConn
	require.Eventually(t, func() bool {
		server.connMu.Lock()
		defer server.connMu.Unlock()
		conn = server.conn
		return conn != nil
	}, time.Second, time.Millisecond)
	return conn
}
