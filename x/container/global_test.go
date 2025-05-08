package container

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobal_SetAndGet(t *testing.T) {
	Set("key", "value")
	v, err := Get[string]("key")
	require.NoError(t, err)
	assert.Equal(t, "value", v)

	non, err := Get[int]("key-unknown")
	require.ErrorIs(t, err, ErrKeyNotFound)
	assert.Equal(t, 0, non)

	v2, err := Get[int]("key")
	require.ErrorIs(t, err, ErrTypeMismatch)
	assert.Equal(t, 0, v2)

	assert.Equal(t, "value", MustGet[string]("key"))
	assert.Panics(t, func() {
		MustGet[int]("key")
	})
}
