package filesystem

import (
	"context"
	"errors"
	"time"
)

var ErrNotSupported = errors.New("filesystem: the operation is not supported")

type Filesystem interface {
	// Read the value at the given path.
	Read(ctx context.Context, path string) ([]byte, error)

	// Write the value at the given path.
	Write(ctx context.Context, path string, value []byte) error

	// Delete the value at the given path.
	Delete(ctx context.Context, path string) error

	// Exists checks if the path exists.
	Exists(ctx context.Context, path string) (bool, error)

	// Rename renames the value from the old path to the new path.
	Rename(ctx context.Context, oldPath, newPath string) error

	// Link creates a hard link from the old path to the new path.
	Link(ctx context.Context, oldPath, newPath string) error

	// Symlink creates a symbolic link from the old path to the new path.
	Symlink(ctx context.Context, oldPath, newPath string) error

	// Files lists the files in the given path.
	Files(ctx context.Context, path string) ([]string, error)

	// AllFiles lists all the files in the given path.(including subdirectories)
	AllFiles(ctx context.Context, path string) ([]string, error)

	// Directories lists the directories in the given path.
	Directories(ctx context.Context, path string) ([]string, error)

	// AllDirectories lists all the directories in the given path.(including subdirectories)
	AllDirectories(ctx context.Context, path string) ([]string, error)

	// MakeDirectory creates a directory at the given path.
	MakeDirectory(ctx context.Context, path string) error

	// DeleteDirectory deletes the directory at the given path.
	DeleteDirectory(ctx context.Context, path string) error

	// IsFile checks if the path is a file.
	IsFile(ctx context.Context, path string) (bool, error)

	// IsDir checks if the path is a directory.
	IsDir(ctx context.Context, path string) (bool, error)

	// Size returns the size of the file in bytes.
	Size(ctx context.Context, path string) (int64, error)

	// LastModified returns the last modified time of the file.
	LastModified(ctx context.Context, path string) (*time.Time, error)

	// Path returns the full path for the given path.
	Path(ctx context.Context, path string) string

	// Name returns the name of the file, without the extension.
	Name(ctx context.Context, path string) string

	// Basename returns the base name of the file, with the extension.
	Basename(ctx context.Context, path string) string

	// Dirname returns the directory name of the file.
	Dirname(ctx context.Context, path string) string

	// Extension returns the extension of the file.
	Extension(ctx context.Context, path string) string
}

type Copyable interface {
	Copy(ctx context.Context, oldPath, newPath string) error
}

type NoopFilesystem struct{}

var (
	_ Filesystem = (*NoopFilesystem)(nil)
	_ Copyable   = (*NoopFilesystem)(nil)
)

func (NoopFilesystem) Read(context.Context, string) ([]byte, error)             { return nil, nil }
func (NoopFilesystem) Write(context.Context, string, []byte) error              { return nil }
func (NoopFilesystem) Delete(context.Context, string) error                     { return nil }
func (NoopFilesystem) Exists(context.Context, string) (bool, error)             { return true, nil }
func (NoopFilesystem) Rename(context.Context, string, string) error             { return nil }
func (NoopFilesystem) Link(context.Context, string, string) error               { return nil }
func (NoopFilesystem) Symlink(context.Context, string, string) error            { return nil }
func (NoopFilesystem) Files(context.Context, string) ([]string, error)          { return nil, nil }
func (NoopFilesystem) AllFiles(context.Context, string) ([]string, error)       { return nil, nil }
func (NoopFilesystem) Directories(context.Context, string) ([]string, error)    { return nil, nil }
func (NoopFilesystem) AllDirectories(context.Context, string) ([]string, error) { return nil, nil }
func (NoopFilesystem) MakeDirectory(context.Context, string) error              { return nil }
func (NoopFilesystem) DeleteDirectory(context.Context, string) error            { return nil }
func (NoopFilesystem) IsFile(context.Context, string) (bool, error)             { return false, nil }
func (NoopFilesystem) IsDir(context.Context, string) (bool, error)              { return false, nil }
func (NoopFilesystem) Size(context.Context, string) (int64, error)              { return 0, nil }
func (NoopFilesystem) LastModified(context.Context, string) (*time.Time, error) { return nil, nil }
func (NoopFilesystem) Path(context.Context, string) string                      { return "" }
func (NoopFilesystem) Name(context.Context, string) string                      { return "" }
func (NoopFilesystem) Basename(context.Context, string) string                  { return "" }
func (NoopFilesystem) Dirname(context.Context, string) string                   { return "" }
func (NoopFilesystem) Extension(context.Context, string) string                 { return "" }
func (NoopFilesystem) Copy(context.Context, string, string) error               { return nil }
