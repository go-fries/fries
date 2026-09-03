package local

import (
	"testing"

	"github.com/go-fries/fries/filesystem/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilesystemStatRejectsInvalidPath(t *testing.T) {
	storage, err := New(t.TempDir())
	require.NoError(t, err)

	_, err = storage.Stat(t.Context(), "../escape")

	assert.ErrorIs(t, err, filesystem.ErrInvalidPath)
}
