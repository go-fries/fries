package env

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	ctx := t.Context()

	env1, ok1 := FromContext(ctx)
	assert.NotEqual(t, Dev, env1)
	assert.False(t, ok1)
	assert.Equal(t, Env(""), env1)

	envCtx := NewContext(ctx, Dev)
	env2, ok2 := FromContext(envCtx)
	assert.Equal(t, Dev, env2)
	assert.True(t, ok2)
}
