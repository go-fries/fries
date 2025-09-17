package signal

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	buffer bytes.Buffer
	mu     sync.Mutex
)

func TestServer(t *testing.T) {
	srv := newServer()

	go srv.Start(t.Context()) //nolint:errcheck

	time.Sleep(2 * time.Second)

	assert.NoError(t, syscall.Kill(os.Getpid(), syscall.SIGUSR1))
	assert.NoError(t, syscall.Kill(os.Getpid(), syscall.SIGUSR2))

	time.Sleep(2 * time.Second)

	mu.Lock()
	assert.Equal(t, `exampleHandler signal: user defined signal 1
signal: user defined signal 1, handler: *signal.example2Handler, err: example2Handler panic
exampleHandler signal: user defined signal 2
`, buffer.String())
	mu.Unlock()

	srv.Stop(t.Context()) //nolint:errcheck
}

func newServer() *Server {
	srv := NewServer(
		WithRecovery(func(err any, signal os.Signal, handler Handler) {
			mu.Lock()
			defer mu.Unlock()
			buffer.WriteString(fmt.Sprintf("signal: %s, handler: %T, err: %v\n", signal, handler, err))
		}),
	)

	srv.Register(&exampleHandler{}, &example2Handler{})

	return srv
}

type exampleHandler struct{}

func (h *exampleHandler) Listen() []os.Signal {
	return []os.Signal{syscall.SIGUSR1, syscall.SIGUSR2}
}

func (h *exampleHandler) Handle(sig os.Signal) {
	mu.Lock()
	defer mu.Unlock()
	buffer.WriteString(fmt.Sprintf("exampleHandler signal: %s\n", sig))
}

type example2Handler struct{}

func (h *example2Handler) Listen() []os.Signal {
	return []os.Signal{syscall.SIGUSR1}
}

func (h *example2Handler) Async() bool {
	return false
}

func (h *example2Handler) Handle(os.Signal) {
	panic("example2Handler panic")
}
