package jet

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContext_Client(t *testing.T) {
	ctx := context.Background()

	got1, ok1 := ClientFromContext(ctx)
	assert.False(t, ok1)
	assert.Nil(t, got1)

	client := &Client{}
	ctx = ContextWithClient(context.Background(), client)

	got2, ok2 := ClientFromContext(ctx)
	assert.True(t, ok2)
	assert.Equal(t, client, got2)
}
