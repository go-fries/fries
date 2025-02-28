package event

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	d1, ok1 := FromContext(context.Background())
	assert.False(t, ok1)
	assert.Nil(t, d1)

	var d *Dispatcher
	ctx := NewContext(context.Background(), d)

	d2, ok2 := FromContext(ctx)
	assert.True(t, ok2)
	assert.Equal(t, d, d2)
}
