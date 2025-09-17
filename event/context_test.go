package event

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	d1, ok1 := FromContext(t.Context())
	assert.False(t, ok1)
	assert.Nil(t, d1)

	var d *Dispatcher
	ctx := NewContext(t.Context(), d)

	d2, ok2 := FromContext(ctx)
	assert.True(t, ok2)
	assert.Equal(t, d, d2)
}
