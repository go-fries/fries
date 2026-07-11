package filesystem

import (
	"context"
	"errors"
	"io"
	"time"
)

var (
	// ErrInvalidPath indicates that a path does not follow the filesystem's
	// logical path contract.
	ErrInvalidPath = errors.New("filesystem: invalid path")
	// ErrNotFound indicates that the requested file or object does not exist.
	// Drivers wrap ErrNotFound when Open or Stat cannot find a path. Callers
	// should use errors.Is to test for it. Delete does not return ErrNotFound,
	// because deletion is idempotent.
	ErrNotFound = errors.New("filesystem: entry not found")
	// ErrUnsupported indicates that an optional capability or driver-specific
	// extension cannot perform an operation. It is reserved for optional and
	// future APIs; the core Driver contract does not currently return it.
	ErrUnsupported = errors.New("filesystem: operation not supported")
)

// EntryKind identifies the logical kind of a filesystem entry.
type EntryKind uint8

const (
	// EntryKindUnknown indicates that an entry's kind is not known.
	EntryKindUnknown EntryKind = iota
	// EntryKindFile identifies a file or object.
	EntryKindFile
	// EntryKindDirectory identifies a real or virtual directory.
	EntryKindDirectory
)

// Entry describes a file, object, or directory returned by Stat or ListFiles.
// Fields unsupported by a backend use their zero values. For directories, only
// Path and Kind are portable across drivers; other fields may be zero or contain
// backend-specific information and must not be required for portable behavior.
type Entry struct {
	Path         string
	Kind         EntryKind
	Size         int64
	LastModified time.Time
	ContentType  string
	Metadata     map[string]string
}

// IsFile reports whether e represents a file or object.
func (e Entry) IsFile() bool {
	return e.Kind == EntryKindFile
}

// IsDir reports whether e represents a real or virtual directory.
func (e Entry) IsDir() bool {
	return e.Kind == EntryKindDirectory
}

// PutOptions configures a Put operation.
type PutOptions struct {
	ContentType string
	Metadata    map[string]string
}

const (
	// DefaultListLimit is the page size used when ListOptions.Limit is not set.
	DefaultListLimit = 1000
	// MaxListLimit is the largest page size accepted by ListFiles.
	MaxListLimit = 1000
)

// ListOptions configures a ListFiles operation.
type ListOptions struct {
	// Recursive includes files below nested directories or prefixes.
	Recursive bool
	// Limit caps the number of entries returned. A page may contain fewer entries,
	// including none, even when more pages remain. Values less than one use
	// DefaultListLimit; values greater than MaxListLimit use MaxListLimit.
	Limit int
	// Cursor continues a previous listing. Cursors are opaque and scoped to a
	// driver, path, and option set.
	Cursor string
}

// Normalize returns a copy of o with the shared page-size rules applied.
func (o ListOptions) Normalize() ListOptions {
	switch {
	case o.Limit < 1:
		o.Limit = DefaultListLimit
	case o.Limit > MaxListLimit:
		o.Limit = MaxListLimit
	}
	return o
}

// ListPage is one page of a file or object listing.
type ListPage struct {
	Entries []Entry
	// NextCursor continues the listing. Only an empty NextCursor indicates that
	// the listing is complete; Entries may be empty while NextCursor is non-empty.
	NextCursor string
}

// Driver is the minimal storage contract shared by local filesystems and
// object-storage backends.
type Driver interface {
	// Open opens path for streaming reads. The caller must close the result.
	// The logical root "." is not a valid file or object path.
	// If path does not exist, Open returns an error wrapping ErrNotFound.
	Open(ctx context.Context, path string) (io.ReadCloser, error)
	// Put writes src to path, replacing an existing entry. The logical root "."
	// is not a valid file or object path.
	Put(ctx context.Context, path string, src io.Reader, options PutOptions) error
	// Delete removes a file or object. Delete is idempotent: deleting a path
	// that does not exist returns nil. The logical root "." cannot be deleted.
	Delete(ctx context.Context, path string) error
	// Stat returns metadata for a file or object. Stat(".") returns a synthetic
	// directory entry for the logical root. If path does not exist, Stat returns
	// an error wrapping ErrNotFound. Object-storage drivers may perform an
	// additional prefix listing to distinguish a missing object from a virtual
	// directory.
	Stat(ctx context.Context, path string) (Entry, error)
	// ListFiles returns one page of files or objects below a directory or object
	// prefix. Directories, virtual prefixes, and object-storage directory markers
	// are not returned. Use "." to list the logical root. If path has no matching
	// files, including when it does not exist or names a file, ListFiles returns an
	// empty page and a nil error. Entries are sorted by logical path within each
	// page. A page may be empty while NextCursor is non-empty; callers must
	// continue until NextCursor is empty. Ordering across pages follows the
	// backend cursor and is not guaranteed.
	ListFiles(ctx context.Context, path string, options ListOptions) (ListPage, error)
}

// Copier is implemented by drivers with a native copy operation.
type Copier interface {
	Copy(ctx context.Context, src, dst string) error
}

// Mover is implemented by drivers with a native move operation.
// A move is not necessarily atomic on object-storage backends.
type Mover interface {
	Move(ctx context.Context, src, dst string) error
}

// Linker is implemented by drivers that support hard links.
type Linker interface {
	Link(ctx context.Context, src, dst string) error
}

// Symlinker is implemented by drivers that support symbolic links.
type Symlinker interface {
	Symlink(ctx context.Context, target, link string) error
}

// DirectoryManager is implemented by drivers with real directory semantics.
type DirectoryManager interface {
	MakeDirectory(ctx context.Context, path string) error
	DeleteDirectory(ctx context.Context, path string) error
}
