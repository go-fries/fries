package signal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockAsyncHandler struct {
	AsyncHandler
}

func TestAsyncHandler(t *testing.T) {
	assert.Implements(t, (*AsyncHandler)(nil), (*mockAsyncHandler)(nil))
}
