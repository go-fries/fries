package local

import (
	"io"
	"runtime"
	"strings"
	"testing"

	"github.com/go-fries/fries/filesystem/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilesystem(t *testing.T) {
	storage, err := New(t.TempDir())
	require.NoError(t, err)
	ctx := t.Context()

	require.NoError(t, storage.MakeDirectory(ctx, "dir/nested"))
	require.NoError(t, storage.Put(
		ctx,
		"dir/file.txt",
		strings.NewReader("content"),
		filesystem.PutOptions{},
	))
	require.NoError(t, storage.Put(
		ctx,
		"dir/nested/child.txt",
		strings.NewReader("child"),
		filesystem.PutOptions{},
	))

	reader, err := storage.Open(ctx, "dir/file.txt")
	require.NoError(t, err)
	value, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	assert.Equal(t, "content", string(value))

	entry, err := storage.Stat(ctx, "dir/file.txt")
	require.NoError(t, err)
	assert.True(t, entry.IsFile())
	assert.EqualValues(t, len("content"), entry.Size)

	page, err := storage.List(ctx, "dir", filesystem.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, []filesystem.Entry{
		{Path: "dir/file.txt", Kind: filesystem.EntryKindFile, Size: 7, LastModified: page.Entries[0].LastModified},
		{Path: "dir/nested", Kind: filesystem.EntryKindDirectory, Size: page.Entries[1].Size, LastModified: page.Entries[1].LastModified},
	}, page.Entries)

	recursive, err := storage.List(ctx, "dir", filesystem.ListOptions{Recursive: true})
	require.NoError(t, err)
	assert.Equal(t, []string{"dir/file.txt", "dir/nested", "dir/nested/child.txt"}, entryPaths(recursive.Entries))

	files, err := storage.List(ctx, ".", filesystem.ListOptions{
		Recursive: true,
		Kind:      filesystem.EntryKindFile,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"dir/file.txt", "dir/nested/child.txt"}, entryPaths(files.Entries))

	require.NoError(t, storage.Move(ctx, "dir/file.txt", "dir/moved.txt"))
	require.NoError(t, storage.Link(ctx, "dir/moved.txt", "dir/link.txt"))

	if runtime.GOOS != "windows" {
		require.NoError(t, storage.Symlink(ctx, "dir/moved.txt", "dir/symlink.txt"))
		symlink, err := storage.Open(ctx, "dir/symlink.txt")
		require.NoError(t, err)
		value, err := io.ReadAll(symlink)
		require.NoError(t, err)
		require.NoError(t, symlink.Close())
		assert.Equal(t, "content", string(value))
	}

	require.NoError(t, storage.Delete(ctx, "dir/link.txt"))
	require.NoError(t, storage.DeleteDirectory(ctx, "dir"))
	_, err = storage.Stat(ctx, "dir")
	assert.ErrorIs(t, err, filesystem.ErrNotFound)
}

func TestFilesystemListPagination(t *testing.T) {
	storage, err := New(t.TempDir())
	require.NoError(t, err)
	ctx := t.Context()

	for _, path := range []string{"a.txt", "b.txt", "c.txt"} {
		require.NoError(t, storage.Put(ctx, path, strings.NewReader(path), filesystem.PutOptions{}))
	}

	first, err := storage.List(ctx, ".", filesystem.ListOptions{Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, []string{"a.txt", "b.txt"}, entryPaths(first.Entries))
	assert.Equal(t, "b.txt", first.NextCursor)

	second, err := storage.List(ctx, ".", filesystem.ListOptions{Limit: 2, Cursor: first.NextCursor})
	require.NoError(t, err)
	assert.Equal(t, []string{"c.txt"}, entryPaths(second.Entries))
	assert.Empty(t, second.NextCursor)
}

func TestFilesystemRejectsEscapingPaths(t *testing.T) {
	storage, err := New(t.TempDir())
	require.NoError(t, err)

	err = storage.Put(t.Context(), "../escape.txt", strings.NewReader("escape"), filesystem.PutOptions{})
	assert.ErrorIs(t, err, filesystem.ErrInvalidPath)
	assert.ErrorIs(t, storage.DeleteDirectory(t.Context(), "."), filesystem.ErrInvalidPath)
}

func TestFilesystemMissingEntry(t *testing.T) {
	storage, err := New(t.TempDir())
	require.NoError(t, err)

	_, err = storage.Open(t.Context(), "missing.txt")
	assert.ErrorIs(t, err, filesystem.ErrNotFound)
	_, err = storage.Stat(t.Context(), "missing.txt")
	assert.ErrorIs(t, err, filesystem.ErrNotFound)
}

func entryPaths(entries []filesystem.Entry) []string {
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		paths = append(paths, entry.Path)
	}
	return paths
}
