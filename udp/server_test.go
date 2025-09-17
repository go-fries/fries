package udp

import (
	"net"
	"sync"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	var (
		server *Server
		wg     sync.WaitGroup
		done   = make(chan []byte, 1)
	)

	wg.Add(2)

	go func() {
		defer wg.Done()

		server = NewServer(":12190", WithHandler(func(msg *Message) {
			done <- msg.Body
		}), WithRecoveryHandler(func(_ *Message, err any) {
			t.Log(err)
		}), WithBufSize(1024))

		go server.Start(t.Context()) //nolint:errcheck

		time.Sleep(time.Second * 5)
		_ = server.Stop(t.Context())
	}()

	go func() {
		defer wg.Done()

		time.Sleep(time.Second * 3)

		c, err := net.Dial("udp", ":12190")
		if err != nil {
			t.Error(err)
			return
		}
		defer c.Close() //nolint:errcheck

		_, err = c.Write([]byte("test"))
		if err != nil {
			t.Error(err)
			return
		}
	}()

	wg.Wait()

	buf := <-done
	if string(buf) != "test" {
		t.Fatal("buf not equal test")
	}
}
