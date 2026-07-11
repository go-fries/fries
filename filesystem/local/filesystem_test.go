package local

import (
	"io"
	"os"
	"path/filepath"
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

	page, err := storage.ListFiles(ctx, "dir", filesystem.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, []filesystem.Entry{
		{Path: "dir/file.txt", Kind: filesystem.EntryKindFile, Size: 7, LastModified: page.Entries[0].LastModified},
	}, page.Entries)

	recursive, err := storage.ListFiles(ctx, "dir", filesystem.ListOptions{Recursive: true})
	require.NoError(t, err)
	assert.Equal(t, []string{"dir/file.txt", "dir/nested/child.txt"}, entryPaths(recursive.Entries))

	files, err := storage.ListFiles(ctx, ".", filesystem.ListOptions{Recursive: true})
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

	first, err := storage.ListFiles(ctx, ".", filesystem.ListOptions{Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, []string{"a.txt", "b.txt"}, entryPaths(first.Entries))
	assert.Equal(t, "b.txt", first.NextCursor)

	second, err := storage.ListFiles(ctx, ".", filesystem.ListOptions{Limit: 2, Cursor: first.NextCursor})
	require.NoError(t, err)
	assert.Equal(t, []string{"c.txt"}, entryPaths(second.Entries))
	assert.Empty(t, second.NextCursor)
}

func TestFilesystemPutContentLength(t *testing.T) {
	storage, err := New(t.TempDir())
	require.NoError(t, err)
	source := struct{ io.Reader }{Reader: strings.NewReader("content")}

	err = storage.Put(t.Context(), "file.txt", source, filesystem.PutOptions{})
	assert.ErrorIs(t, err, filesystem.ErrContentLengthRequired)

	length := int64(len("content"))
	err = storage.Put(t.Context(), "file.txt", source, filesystem.PutOptions{
		ContentLength: &length,
	})
	require.NoError(t, err)

	reader, err := storage.Open(t.Context(), "file.txt")
	require.NoError(t, err)
	value, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	assert.Equal(t, "content", string(value))
}

func TestFilesystemListWithoutMatches(t *testing.T) {
	storage, err := New(t.TempDir())
	require.NoError(t, err)
	ctx := t.Context()

	missing, err := storage.ListFiles(ctx, "missing", filesystem.ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, missing.Entries)
	assert.Empty(t, missing.NextCursor)

	require.NoError(t, storage.Put(ctx, "file.txt", strings.NewReader("content"), filesystem.PutOptions{}))
	file, err := storage.ListFiles(ctx, "file.txt", filesystem.ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, file.Entries)
	assert.Empty(t, file.NextCursor)
}

func TestFilesystemRejectsEscapingPaths(t *testing.T) {
	storage, err := New(t.TempDir())
	require.NoError(t, err)

	err = storage.Put(t.Context(), "../escape.txt", strings.NewReader("escape"), filesystem.PutOptions{})
	assert.ErrorIs(t, err, filesystem.ErrInvalidPath)
	_, err = storage.Open(t.Context(), ".")
	assert.ErrorIs(t, err, filesystem.ErrInvalidPath)
	err = storage.Put(t.Context(), ".", strings.NewReader("root"), filesystem.PutOptions{})
	assert.ErrorIs(t, err, filesystem.ErrInvalidPath)
	assert.ErrorIs(t, storage.Delete(t.Context(), "."), filesystem.ErrInvalidPath)
	assert.ErrorIs(t, storage.DeleteDirectory(t.Context(), "."), filesystem.ErrInvalidPath)
}

func TestFilesystemRejectsEscapingSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic-link creation may require additional privileges")
	}

	root := t.TempDir()
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "outside.txt"), []byte("outside"), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(outside, "directory"), 0o755))
	require.NoError(t, os.Symlink(outside, filepath.Join(root, "escape")))
	require.NoError(t, os.WriteFile(filepath.Join(root, "inside.txt"), []byte("inside"), 0o644))

	storage, err := New(root)
	require.NoError(t, err)
	ctx := t.Context()

	_, err = storage.Open(ctx, "escape/outside.txt")
	assert.Error(t, err)
	assert.Error(t, storage.Put(
		ctx,
		"escape/created.txt",
		strings.NewReader("created"),
		filesystem.PutOptions{},
	))
	_, err = os.Stat(filepath.Join(outside, "created.txt"))
	assert.ErrorIs(t, err, os.ErrNotExist)

	_, err = storage.Stat(ctx, "escape/outside.txt")
	assert.Error(t, err)
	_, err = storage.ListFiles(ctx, "escape", filesystem.ListOptions{})
	assert.Error(t, err)

	assert.Error(t, storage.Delete(ctx, "escape/outside.txt"))
	_, err = os.Stat(filepath.Join(outside, "outside.txt"))
	assert.NoError(t, err)

	assert.Error(t, storage.Move(ctx, "inside.txt", "escape/moved.txt"))
	_, err = os.Stat(filepath.Join(root, "inside.txt"))
	assert.NoError(t, err)
	assert.Error(t, storage.Link(ctx, "inside.txt", "escape/linked.txt"))
	assert.Error(t, storage.Symlink(ctx, "inside.txt", "escape/symlink.txt"))
	assert.Error(t, storage.MakeDirectory(ctx, "escape/created-directory"))
	assert.Error(t, storage.DeleteDirectory(ctx, "escape/directory"))
	_, err = os.Stat(filepath.Join(outside, "directory"))
	assert.NoError(t, err)
}

func TestFilesystemStatRoot(t *testing.T) {
	storage, err := New(t.TempDir())
	require.NoError(t, err)

	entry, err := storage.Stat(t.Context(), ".")
	require.NoError(t, err)
	assert.Equal(t, filesystem.Entry{Path: ".", Kind: filesystem.EntryKindDirectory}, entry)
}

func TestFilesystemMissingEntry(t *testing.T) {
	storage, err := New(t.TempDir())
	require.NoError(t, err)

	_, err = storage.Open(t.Context(), "missing.txt")
	assert.ErrorIs(t, err, filesystem.ErrNotFound)
	_, err = storage.Stat(t.Context(), "missing.txt")
	assert.ErrorIs(t, err, filesystem.ErrNotFound)
	assert.NoError(t, storage.Delete(t.Context(), "missing.txt"))
}

func entryPaths(entries []filesystem.Entry) []string {
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		paths = append(paths, entry.Path)
	}
	return paths
}
