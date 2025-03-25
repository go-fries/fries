package local

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

func TestFilesystem_basic(t *testing.T) {
	// init
	require.NoError(t, os.Mkdir("./testfile/basic", os.ModePerm))
	defer t.Cleanup(func() {
		assert.NoError(t, os.RemoveAll("./testfile/basic"))
	})

	// local
	local := NewStorage("./testfile/basic")
	filename := "test.txt"

	// write
	assert.NoError(t, local.Write(ctx, filename, []byte("test")))

	// read
	data, err := local.Read(ctx, filename)
	assert.NoError(t, err)
	assert.Equal(t, []byte("test"), data)

	data, err = local.Read(ctx, "missing")
	assert.Error(t, err)
	assert.Nil(t, data)

	// has
	has, err := local.Exists(ctx, filename)
	assert.NoError(t, err)
	assert.True(t, has)

	missing, err := local.Exists(ctx, "missing")
	assert.NoError(t, err)
	assert.False(t, missing)

	// move
	assert.NoError(t, local.Rename(ctx, filename, "test2.txt"))
	has, err = local.Exists(ctx, filename)
	assert.NoError(t, err)
	assert.False(t, has)
	has, err = local.Exists(ctx, "test2.txt")
	assert.NoError(t, err)
	assert.True(t, has)

	// link
	assert.NoError(t, local.Link(ctx, "test2.txt", "test4.txt"))
	has, err = local.Exists(ctx, "test4.txt")
	assert.NoError(t, err)
	assert.True(t, has)
	data, err = local.Read(ctx, "test4.txt")
	assert.NoError(t, err)
	assert.Equal(t, []byte("test"), data)

	// symlink
	// assert.NoError(t, local.Symlink(ctx, "test2.txt", "test5.txt"))
	// data, err = local.Read(ctx, "test5.txt")
	// assert.NoError(t, err)
	// assert.Equal(t, []byte("test"), data)

	// path
	path := local.Path(ctx, "1.jpg")
	assert.Equal(t, "./testfile/basic/1.jpg", path)

	// delete
	assert.NoError(t, local.Delete(ctx, "test4.txt"))
	has, err = local.Exists(ctx, "test4.txt")
	assert.NoError(t, err)
	assert.False(t, has)
}

func TestFilesystem_Path(t *testing.T) {
	local := NewStorage("./testfile/path")

	tests := []struct {
		name string
		path string
		want string
	}{
		{"", ".jpg", "./testfile/path/.jpg"},
		{"", "", "./testfile/path/"},
		{"", "1.jpg", "./testfile/path/1.jpg"},
		{"", "2.jpg", "./testfile/path/2.jpg"},
		{"", "2", "./testfile/path/2"},
		{"", "2/3", "./testfile/path/2/3"},
		{"", "/4/3.jpg", "./testfile/path/4/3.jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, local.Path(ctx, tt.path))
		})
	}
}

func TestFilesystem_Name(t *testing.T) {
	local := NewStorage("./testfile/path")

	tests := []struct {
		name string
		path string
		want string
	}{
		{"", ".jpg", ""},
		{"", "", "path"},
		{"", "/1/", "1"},
		{"", "1.jpg", "1"},
		{"", "2.jpg", "2"},
		{"", "2", "2"},
		{"", "2/3", "3"},
		{"", "/4/3.jpg", "3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, local.Name(ctx, tt.path))
		})
	}
}

func TestFilesystem_Basename(t *testing.T) {
	local := NewStorage("./testfile/path")

	tests := []struct {
		name string
		path string
		want string
	}{
		{"", ".jpg", ".jpg"},
		{"", "/1/", "1"},
		{"", "", "path"},
		{"", "1.jpg", "1.jpg"},
		{"", "2.jpg", "2.jpg"},
		{"", "2", "2"},
		{"", "2/3", "3"},
		{"", "/4/3.jpg", "3.jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, local.Basename(ctx, tt.path))
		})
	}
}

func TestFilesystem_Dirname(t *testing.T) {
	local := NewStorage("./testfile/path")

	tests := []struct {
		name string
		path string
		want string
	}{
		{"", ".jpg", "testfile/path"},
		{"", "", "testfile/path"},
		{"", "1.jpg", "testfile/path"},
		{"", "2.jpg", "testfile/path"},
		{"", "2", "testfile/path"},
		{"", "3/4", "testfile/path/3"},
		{"", "/5/6.jpg", "testfile/path/5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, local.Dirname(ctx, tt.path))
		})
	}
}

func TestFilesystem_Extension(t *testing.T) {
	local := NewStorage("./testfile/path")

	tests := []struct {
		name string
		path string
		want string
	}{
		{"", ".jpg", ".jpg"},
		{"", "", ""},
		{"", "1.jpg", ".jpg"},
		{"", "2.jpg", ".jpg"},
		{"", "2", ""},
		{"", "2/3", ""},
		{"", "/4/3.jpg", ".jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, local.Extension(ctx, tt.path))
		})
	}
}
