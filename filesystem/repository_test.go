package filesystem

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository(t *testing.T) {
	driver := newMemoryDriver()
	repository := NewRepository(driver)
	ctx := t.Context()

	require.NoError(t, repository.WriteFile(ctx, "source.txt", []byte("content"), PutOptions{
		Metadata: map[string]string{"owner": "test"},
	}))

	value, err := repository.ReadFile(ctx, "source.txt")
	require.NoError(t, err)
	assert.Equal(t, []byte("content"), value)

	exists, err := repository.Exists(ctx, "source.txt")
	require.NoError(t, err)
	assert.True(t, exists)

	require.NoError(t, repository.Copy(ctx, "source.txt", "copy.txt"))
	copyEntry, err := repository.Stat(ctx, "copy.txt")
	require.NoError(t, err)
	assert.Equal(t, "test", copyEntry.Metadata["owner"])

	require.NoError(t, repository.Move(ctx, "copy.txt", "moved.txt"))
	exists, err = repository.Exists(ctx, "copy.txt")
	require.NoError(t, err)
	assert.False(t, exists)
	exists, err = repository.Exists(ctx, "moved.txt")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestRepositoryMissingEntry(t *testing.T) {
	repository := NewRepository(newMemoryDriver())

	exists, err := repository.Exists(t.Context(), "missing.txt")
	require.NoError(t, err)
	assert.False(t, exists)

	_, err = repository.ReadFile(t.Context(), "missing.txt")
	assert.ErrorIs(t, err, ErrNotFound)
}

type memoryObject struct {
	value       []byte
	contentType string
	metadata    map[string]string
}

type memoryDriver struct {
	objects map[string]memoryObject
}

func newMemoryDriver() *memoryDriver {
	return &memoryDriver{objects: make(map[string]memoryObject)}
}

func (d *memoryDriver) Open(_ context.Context, path string) (io.ReadCloser, error) {
	object, ok := d.objects[path]
	if !ok {
		return nil, ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(object.value)), nil
}

func (d *memoryDriver) Put(
	_ context.Context,
	path string,
	src io.Reader,
	options PutOptions,
) error {
	value, err := io.ReadAll(src)
	if err != nil {
		return err
	}
	d.objects[path] = memoryObject{
		value:       value,
		contentType: options.ContentType,
		metadata:    cloneMetadata(options.Metadata),
	}
	return nil
}

func (d *memoryDriver) Delete(_ context.Context, path string) error {
	if _, ok := d.objects[path]; !ok {
		return ErrNotFound
	}
	delete(d.objects, path)
	return nil
}

func (d *memoryDriver) Stat(_ context.Context, path string) (Entry, error) {
	object, ok := d.objects[path]
	if !ok {
		return Entry{}, ErrNotFound
	}
	return Entry{
		Path:        path,
		Kind:        EntryKindFile,
		Size:        int64(len(object.value)),
		ContentType: object.contentType,
		Metadata:    cloneMetadata(object.metadata),
	}, nil
}

func (d *memoryDriver) ListFiles(context.Context, string, ListOptions) (ListPage, error) {
	return ListPage{}, nil
}
