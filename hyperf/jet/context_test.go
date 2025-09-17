package jet

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContext_Client(t *testing.T) {
	ctx := t.Context()

	got1, ok1 := ClientFromContext(ctx)
	assert.False(t, ok1)
	assert.Nil(t, got1)

	client := &Client{}
	ctx = ContextWithClient(t.Context(), client)

	got2, ok2 := ClientFromContext(ctx)
	assert.True(t, ok2)
	assert.Equal(t, client, got2)
}
