package filesystem

import "context"

type Repository interface {
	Filesystem
	Copyable

	// Get retrieves the value for the given path.
	Get(ctx context.Context, path string) ([]byte, error)

	// Set sets the value for the given path.
	Set(ctx context.Context, path string, value []byte) error

	// Put sets the value for the given path.
	Put(ctx context.Context, path string, value []byte) error

	// Destroy deletes the value for the given path.
	Destroy(ctx context.Context, path string) error

	// Has checks if the path exists.
	Has(ctx context.Context, path string) (bool, error)

	// Missing checks if the path does not exist.
	Missing(ctx context.Context, path string) (bool, error)

	// Move moves the value from the old path to the new path.
	Move(ctx context.Context, oldPath, newPath string) error

	// Prepend prepends the value to the existing value for the given path.
	Prepend(ctx context.Context, path string, value []byte) error

	// Append appends the value to the existing value for the given path.
	Append(ctx context.Context, path string, value []byte) error
}

type repository struct {
	Filesystem
}

func NewRepository(store Filesystem) Repository {
	return &repository{
		Filesystem: store,
	}
}

func (r *repository) Get(ctx context.Context, path string) ([]byte, error) {
	return r.Read(ctx, path)
}

func (r *repository) Set(ctx context.Context, path string, value []byte) error {
	return r.Write(ctx, path, value)
}

func (r *repository) Put(ctx context.Context, path string, value []byte) error {
	return r.Write(ctx, path, value)
}

func (r *repository) Destroy(ctx context.Context, path string) error {
	return r.Delete(ctx, path)
}

func (r *repository) Has(ctx context.Context, path string) (bool, error) {
	return r.Exists(ctx, path)
}

func (r *repository) Missing(ctx context.Context, path string) (bool, error) {
	had, err := r.Exists(ctx, path)
	if err != nil {
		return false, err
	}

	return !had, nil
}

func (r *repository) Move(ctx context.Context, oldPath, newPath string) error {
	return r.Rename(ctx, oldPath, newPath)
}

func (r *repository) Prepend(ctx context.Context, path string, value []byte) error {
	old, err := r.Read(ctx, path)
	if err != nil {
		return err
	}
	return r.Write(ctx, path, append(value, old...))
}

func (r *repository) Append(ctx context.Context, path string, value []byte) error {
	old, err := r.Read(ctx, path)
	if err != nil {
		return err
	}
	return r.Write(ctx, path, append(old, value...))
}

func (r *repository) Copy(ctx context.Context, oldPath, newPath string) error {
	if copier, ok := r.Filesystem.(Copyable); ok {
		return copier.Copy(ctx, oldPath, newPath)
	}

	old, err := r.Read(ctx, oldPath)
	if err != nil {
		return err
	}
	return r.Write(ctx, newPath, old)
}
