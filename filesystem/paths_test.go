package filesystem

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: ".", want: true},
		{path: "file.txt", want: true},
		{path: "dir/file.txt", want: true},
		{path: "", want: false},
		{path: "/file.txt", want: false},
		{path: "dir/", want: false},
		{path: "dir//file.txt", want: false},
		{path: "../file.txt", want: false},
		{path: `dir\file.txt`, want: false},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			assert.Equal(t, test.want, ValidPath(test.path))
		})
	}
}

func TestValidatePath(t *testing.T) {
	assert.NoError(t, ValidatePath("dir/file.txt"))
	assert.ErrorIs(t, ValidatePath("../file.txt"), ErrInvalidPath)
}

func TestPathPrefixer(t *testing.T) {
	prefixer := NewPathPrefixer("/tenant/root/")

	assert.Equal(t, "tenant/root", prefixer.Prefix("."))
	assert.Equal(t, "tenant/root/dir/file.txt", prefixer.Prefix("dir/file.txt"))

	path, ok := prefixer.Strip("tenant/root/dir/file.txt")
	assert.True(t, ok)
	assert.Equal(t, "dir/file.txt", path)

	path, ok = prefixer.Strip("tenant/root")
	assert.True(t, ok)
	assert.Equal(t, ".", path)

	_, ok = prefixer.Strip("other/file.txt")
	assert.False(t, ok)
}

func TestValidatePathError(t *testing.T) {
	err := ValidatePath("../file.txt")
	assert.True(t, errors.Is(err, ErrInvalidPath))
}
