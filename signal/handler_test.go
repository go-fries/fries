package signal

import (
	"context"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockAsyncHandler struct {
	AsyncHandler
}

func (h *mockAsyncHandler) Listen() []os.Signal {
	return []os.Signal{syscall.SIGUSR1}
}

func (h *mockAsyncHandler) Handle(context.Context, os.Signal) {}

func TestAsyncHandler(t *testing.T) {
	assert.Implements(t, (*AsyncHandler)(nil), (*mockAsyncHandler)(nil))
	assert.Implements(t, (*Handler)(nil), (*mockAsyncHandler)(nil))
}
