package signal

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandlerFunc(t *testing.T) {
	ctx := t.Context()
	wantErr := errors.New("reload failed")
	handler := HandlerFunc(func(gotCtx context.Context, sig os.Signal) error {
		assert.Same(t, ctx, gotCtx)
		assert.Equal(t, os.Interrupt, sig)
		return wantErr
	})
	assert.Implements(t, (*Handler)(nil), handler)
	assert.ErrorIs(t, handler.Handle(ctx, os.Interrupt), wantErr)
}
