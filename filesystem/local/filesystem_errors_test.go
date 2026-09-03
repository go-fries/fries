package local

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/go-fries/fries/filesystem/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilesystemOperationsHonorCanceledContext(t *testing.T) {
	storage, err := New(t.TempDir())
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	for _, operation := range filesystemOperations(ctx, storage) {
		t.Run(operation.name, func(t *testing.T) {
			assert.ErrorIs(t, operation.run(), context.Canceled)
		})
	}
}

func TestFilesystemOperationsRejectInvalidPaths(t *testing.T) {
	storage, err := New(t.TempDir())
	require.NoError(t, err)
	ctx := t.Context()

	operations := []filesystemOperation{
		{name: "list", run: func() error {
			_, err := storage.ListFiles(ctx, "../escape", filesystem.ListOptions{})
			return err
		}},
		{name: "move source", run: func() error {
			return storage.Move(ctx, "../source", "destination.txt")
		}},
		{name: "move destination", run: func() error {
			return storage.Move(ctx, "source.txt", "../destination")
		}},
		{name: "link source", run: func() error {
			return storage.Link(ctx, "../source", "link.txt")
		}},
		{name: "link destination", run: func() error {
			return storage.Link(ctx, "source.txt", "../link")
		}},
		{name: "symlink target", run: func() error {
			return storage.Symlink(ctx, "../target", "link.txt")
		}},
		{name: "symlink path", run: func() error {
			return storage.Symlink(ctx, "target.txt", "../link")
		}},
		{name: "make directory", run: func() error {
			return storage.MakeDirectory(ctx, "../directory")
		}},
		{name: "delete directory", run: func() error {
			return storage.DeleteDirectory(ctx, "../directory")
		}},
	}

	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			assert.ErrorIs(t, operation.run(), filesystem.ErrInvalidPath)
		})
	}
}

func TestFilesystemOperationsReturnMissingRootError(t *testing.T) {
	storage, err := New(filepath.Join(t.TempDir(), "missing"))
	require.NoError(t, err)
	ctx := t.Context()

	for _, operation := range filesystemOperations(ctx, storage) {
		t.Run(operation.name, func(t *testing.T) {
			assert.ErrorIs(t, operation.run(), filesystem.ErrNotFound)
		})
	}
}

func TestFilesystemPutHandlesReaderEdgeCases(t *testing.T) {
	t.Run("nil reader", func(t *testing.T) {
		root := t.TempDir()
		storage, err := New(root)
		require.NoError(t, err)

		require.NoError(t, storage.Put(t.Context(), "empty.txt", nil, filesystem.PutOptions{}))

		content, err := os.ReadFile(filepath.Join(root, "empty.txt"))
		require.NoError(t, err)
		assert.Empty(t, content)
	})

	t.Run("reader error", func(t *testing.T) {
		storage, err := New(t.TempDir())
		require.NoError(t, err)
		sentinel := errors.New("read failed")
		length := int64(1)

		err = storage.Put(
			t.Context(),
			"file.txt",
			errorReader{err: sentinel},
			filesystem.PutOptions{ContentLength: &length},
		)

		assert.ErrorIs(t, err, sentinel)
	})

	t.Run("short reader", func(t *testing.T) {
		storage, err := New(t.TempDir())
		require.NoError(t, err)
		length := int64(len("content") + 1)
		source := struct{ io.Reader }{Reader: strings.NewReader("content")}

		err = storage.Put(
			t.Context(),
			"file.txt",
			source,
			filesystem.PutOptions{ContentLength: &length},
		)

		assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	})

	t.Run("missing parent directory", func(t *testing.T) {
		storage, err := New(t.TempDir())
		require.NoError(t, err)

		err = storage.Put(
			t.Context(),
			"missing/file.txt",
			strings.NewReader("content"),
			filesystem.PutOptions{},
		)

		assert.ErrorIs(t, err, filesystem.ErrNotFound)
	})
}

func TestFilesystemDeleteReturnsPathErrorForNonemptyDirectory(t *testing.T) {
	storage, err := New(t.TempDir())
	require.NoError(t, err)
	require.NoError(t, storage.MakeDirectory(t.Context(), "directory"))
	require.NoError(t, storage.Put(
		t.Context(),
		"directory/file.txt",
		strings.NewReader("content"),
		filesystem.PutOptions{},
	))

	err = storage.Delete(t.Context(), "directory")

	var pathError *fs.PathError
	require.ErrorAs(t, err, &pathError)
	assert.Equal(t, "delete", pathError.Op)
	assert.Equal(t, "directory", pathError.Path)
}

func TestListEntriesPropagatesFilesystemErrors(t *testing.T) {
	sentinel := errors.New("filesystem failed")
	mapFS := fstest.MapFS{"file.txt": &fstest.MapFile{Data: []byte("content")}}
	rootInfo, err := fs.Stat(mapFS, ".")
	require.NoError(t, err)
	entries, err := fs.ReadDir(mapFS, ".")
	require.NoError(t, err)

	t.Run("stat root", func(t *testing.T) {
		storage := stubFS{
			stat: func(string) (fs.FileInfo, error) {
				return nil, sentinel
			},
		}

		_, err := listEntries(t.Context(), storage, ".", false)

		assert.ErrorIs(t, err, sentinel)
	})

	t.Run("read directory", func(t *testing.T) {
		storage := stubFS{
			stat: func(string) (fs.FileInfo, error) {
				return rootInfo, nil
			},
			readDir: func(string) ([]fs.DirEntry, error) {
				return nil, sentinel
			},
		}

		_, err := listEntries(t.Context(), storage, ".", false)

		assert.ErrorIs(t, err, sentinel)
	})

	for _, recursive := range []bool{false, true} {
		t.Run("stat entry recursive="+strconv.FormatBool(recursive), func(t *testing.T) {
			storage := stubFS{
				stat: func(name string) (fs.FileInfo, error) {
					if name == "." {
						return rootInfo, nil
					}
					return nil, sentinel
				},
				readDir: func(string) ([]fs.DirEntry, error) {
					return entries, nil
				},
			}

			_, err := listEntries(t.Context(), storage, ".", recursive)

			assert.ErrorIs(t, err, sentinel)
		})
	}

	t.Run("walk directory", func(t *testing.T) {
		tree := fstest.MapFS{"nested/file.txt": &fstest.MapFile{Data: []byte("content")}}
		rootInfo, err := fs.Stat(tree, ".")
		require.NoError(t, err)
		rootEntries, err := fs.ReadDir(tree, ".")
		require.NoError(t, err)
		storage := stubFS{
			stat: func(string) (fs.FileInfo, error) {
				return rootInfo, nil
			},
			readDir: func(name string) ([]fs.DirEntry, error) {
				if name == "." {
					return rootEntries, nil
				}
				return nil, sentinel
			},
		}

		_, err = listEntries(t.Context(), storage, ".", true)

		assert.ErrorIs(t, err, sentinel)
	})
}

func TestListEntriesHonorsCanceledContext(t *testing.T) {
	storage := fstest.MapFS{"file.txt": &fstest.MapFile{Data: []byte("content")}}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	for _, recursive := range []bool{false, true} {
		t.Run("recursive="+strconv.FormatBool(recursive), func(t *testing.T) {
			_, err := listEntries(ctx, storage, ".", recursive)

			assert.ErrorIs(t, err, context.Canceled)
		})
	}
}

func TestListEntriesSkipsSpecialFiles(t *testing.T) {
	storage := fstest.MapFS{"pipe": &fstest.MapFile{Mode: fs.ModeNamedPipe}}

	for _, recursive := range []bool{false, true} {
		t.Run("recursive="+strconv.FormatBool(recursive), func(t *testing.T) {
			entries, err := listEntries(t.Context(), storage, ".", recursive)

			require.NoError(t, err)
			assert.Empty(t, entries)
		})
	}
}

func TestContextReaderReturnsContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	reader := contextReader{ctx: ctx, reader: strings.NewReader("content")}

	n, err := reader.Read(make([]byte, 1))

	assert.Zero(t, n)
	assert.ErrorIs(t, err, context.Canceled)
}

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}

type filesystemOperation struct {
	name string
	run  func() error
}

func filesystemOperations(ctx context.Context, storage *Filesystem) []filesystemOperation {
	return []filesystemOperation{
		{name: "open", run: func() error {
			_, err := storage.Open(ctx, "file.txt")
			return err
		}},
		{name: "put", run: func() error {
			return storage.Put(ctx, "file.txt", strings.NewReader("content"), filesystem.PutOptions{})
		}},
		{name: "delete", run: func() error {
			return storage.Delete(ctx, "file.txt")
		}},
		{name: "stat", run: func() error {
			_, err := storage.Stat(ctx, "file.txt")
			return err
		}},
		{name: "list", run: func() error {
			_, err := storage.ListFiles(ctx, ".", filesystem.ListOptions{})
			return err
		}},
		{name: "move", run: func() error {
			return storage.Move(ctx, "source.txt", "destination.txt")
		}},
		{name: "link", run: func() error {
			return storage.Link(ctx, "source.txt", "link.txt")
		}},
		{name: "symlink", run: func() error {
			return storage.Symlink(ctx, "target.txt", "link.txt")
		}},
		{name: "make directory", run: func() error {
			return storage.MakeDirectory(ctx, "directory")
		}},
		{name: "delete directory", run: func() error {
			return storage.DeleteDirectory(ctx, "directory")
		}},
	}
}

type stubFS struct {
	stat    func(string) (fs.FileInfo, error)
	readDir func(string) ([]fs.DirEntry, error)
}

func (s stubFS) Open(string) (fs.File, error) {
	return nil, fs.ErrInvalid
}

func (s stubFS) Stat(name string) (fs.FileInfo, error) {
	return s.stat(name)
}

func (s stubFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return s.readDir(name)
}
