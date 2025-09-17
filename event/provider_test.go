package event

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProvider(t *testing.T) {
	d := NewDispatcher()
	p := NewProvider(d)
	p.RegisterListeners(nil)

	ctx, err := p.Bootstrap(t.Context())
	assert.NoError(t, err)

	d1, ok := FromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, d, d1)

	ctx, err = p.Terminate(ctx)
	assert.NoError(t, err)

	d2, ok := FromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, d, d2)
}
