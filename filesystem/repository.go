package filesystem

import (
	"bytes"
	"context"
	"errors"
	"io"
	"maps"
	"net/http"
)

// Repository adds whole-file helpers and portable copy and move fallbacks to a
// Driver.
type Repository struct {
	driver Driver
}

var _ Driver = (*Repository)(nil)

// NewRepository wraps driver with portable convenience operations.
func NewRepository(driver Driver) *Repository {
	return &Repository{driver: driver}
}

// Driver returns the wrapped storage driver.
func (r *Repository) Driver() Driver {
	return r.driver
}

// Open delegates to the wrapped driver.
func (r *Repository) Open(ctx context.Context, path string) (io.ReadCloser, error) {
	return r.driver.Open(ctx, path)
}

// Put delegates to the wrapped driver.
func (r *Repository) Put(
	ctx context.Context,
	path string,
	src io.Reader,
	options PutOptions,
) error {
	return r.driver.Put(ctx, path, src, options)
}

// Delete delegates to the wrapped driver.
func (r *Repository) Delete(ctx context.Context, path string) error {
	return r.driver.Delete(ctx, path)
}

// Stat delegates to the wrapped driver.
func (r *Repository) Stat(ctx context.Context, path string) (Entry, error) {
	return r.driver.Stat(ctx, path)
}

// List delegates to the wrapped driver.
func (r *Repository) List(ctx context.Context, path string, options ListOptions) (ListPage, error) {
	return r.driver.List(ctx, path, options)
}

// ReadFile reads the complete contents of path into memory.
func (r *Repository) ReadFile(ctx context.Context, path string) ([]byte, error) {
	reader, err := r.Open(ctx, path)
	if err != nil {
		return nil, err
	}
	defer reader.Close() //nolint:errcheck

	return io.ReadAll(reader)
}

// WriteFile writes value to path.
func (r *Repository) WriteFile(
	ctx context.Context,
	path string,
	value []byte,
	options PutOptions,
) error {
	if options.ContentType == "" {
		options.ContentType = http.DetectContentType(value)
	}

	return r.Put(ctx, path, bytes.NewReader(value), options)
}

// Exists reports whether path exists.
func (r *Repository) Exists(ctx context.Context, path string) (bool, error) {
	_, err := r.Stat(ctx, path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	return false, err
}

// Copy copies src to dst, preferring a driver's native copy operation.
func (r *Repository) Copy(ctx context.Context, src, dst string) error {
	if copier, ok := r.driver.(Copier); ok {
		return copier.Copy(ctx, src, dst)
	}

	entry, err := r.Stat(ctx, src)
	if err != nil {
		return err
	}
	reader, err := r.Open(ctx, src)
	if err != nil {
		return err
	}
	defer reader.Close() //nolint:errcheck

	return r.Put(ctx, dst, reader, PutOptions{
		ContentType: entry.ContentType,
		Metadata:    cloneMetadata(entry.Metadata),
	})
}

// Move moves src to dst, preferring a driver's native move operation.
func (r *Repository) Move(ctx context.Context, src, dst string) error {
	if mover, ok := r.driver.(Mover); ok {
		return mover.Move(ctx, src, dst)
	}
	if err := r.Copy(ctx, src, dst); err != nil {
		return err
	}
	return r.Delete(ctx, src)
}

func cloneMetadata(metadata map[string]string) map[string]string {
	return maps.Clone(metadata)
}
