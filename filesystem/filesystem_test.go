package filesystem

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStorage(t *testing.T) {
	var (
		noop = NoopFilesystem{}
		ctx  = context.Background()
	)

	_, err := noop.Read(ctx, "test")
	assert.NoError(t, err)

	assert.NoError(t, noop.Write(ctx, "noop", []byte("noop")))
	assert.NoError(t, noop.Delete(ctx, "noop"))

	has, err := noop.Exists(ctx, "noop")
	assert.NoError(t, err)
	assert.True(t, has)

	assert.NoError(t, noop.Rename(ctx, "noop", "noop"))
	assert.NoError(t, noop.Link(ctx, "noop", "noop"))
	assert.NoError(t, noop.Symlink(ctx, "noop", "noop"))

	files, err := noop.Files(ctx, "noop")
	assert.NoError(t, err)
	assert.Len(t, files, 0)

	allFiles, err := noop.AllFiles(ctx, "noop")
	assert.NoError(t, err)
	assert.Len(t, allFiles, 0)

	directories, err := noop.Directories(ctx, "noop")
	assert.NoError(t, err)
	assert.Len(t, directories, 0)

	allDirectories, err := noop.AllDirectories(ctx, "noop")
	assert.NoError(t, err)
	assert.Len(t, allDirectories, 0)

	isFile, err := noop.IsFile(ctx, "noop")
	assert.NoError(t, err)
	assert.False(t, isFile)

	isDir, err := noop.IsDir(ctx, "noop")
	assert.NoError(t, err)
	assert.False(t, isDir)
}
