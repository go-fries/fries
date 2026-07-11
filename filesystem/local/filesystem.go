package local

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	pathpkg "path"
	"path/filepath"
	"sort"

	"github.com/go-fries/fries/filesystem/v4"
)

// Filesystem stores logical filesystem paths below a local root directory.
type Filesystem struct {
	root string
}

var (
	_ filesystem.Driver           = (*Filesystem)(nil)
	_ filesystem.Mover            = (*Filesystem)(nil)
	_ filesystem.Linker           = (*Filesystem)(nil)
	_ filesystem.Symlinker        = (*Filesystem)(nil)
	_ filesystem.DirectoryManager = (*Filesystem)(nil)
)

// New creates a local filesystem rooted at root.
func New(root string) (*Filesystem, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	return &Filesystem{root: filepath.Clean(absolute)}, nil
}

// Open opens path for streaming reads.
func (s *Filesystem) Open(ctx context.Context, path string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := filesystem.ValidateFilePath(path); err != nil {
		return nil, err
	}
	root, err := s.openRoot("open", path)
	if err != nil {
		return nil, err
	}
	defer closeRoot(root)

	reader, err := root.Open(nativePath(path))
	if err != nil {
		return nil, wrapPathError("open", path, err)
	}
	return reader, nil
}

// Put writes src to path, replacing an existing file.
func (s *Filesystem) Put(
	ctx context.Context,
	path string,
	src io.Reader,
	_ filesystem.PutOptions,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := filesystem.ValidateFilePath(path); err != nil {
		return err
	}
	root, err := s.openRoot("put", path)
	if err != nil {
		return err
	}
	defer closeRoot(root)

	file, err := root.OpenFile(nativePath(path), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return wrapPathError("put", path, err)
	}
	_, copyErr := io.Copy(file, contextReader{ctx: ctx, reader: src})
	closeErr := file.Close()
	if copyErr != nil {
		return wrapPathError("put", path, copyErr)
	}
	if closeErr != nil {
		return wrapPathError("put", path, closeErr)
	}
	return nil
}

// Delete removes a file.
func (s *Filesystem) Delete(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := filesystem.ValidateFilePath(path); err != nil {
		return err
	}
	root, err := s.openRoot("delete", path)
	if err != nil {
		return err
	}
	defer closeRoot(root)

	if err := root.Remove(nativePath(path)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return wrapPathError("delete", path, err)
	}
	return nil
}

// Stat returns metadata for path.
func (s *Filesystem) Stat(ctx context.Context, path string) (filesystem.Entry, error) {
	if err := ctx.Err(); err != nil {
		return filesystem.Entry{}, err
	}
	if path == "." {
		return filesystem.Entry{Path: ".", Kind: filesystem.EntryKindDirectory}, nil
	}
	root, err := s.openRoot("stat", path)
	if err != nil {
		return filesystem.Entry{}, err
	}
	defer closeRoot(root)

	info, err := root.Stat(nativePath(path))
	if err != nil {
		return filesystem.Entry{}, wrapPathError("stat", path, err)
	}
	return entryFromInfo(path, info), nil
}

// List returns files below path in lexical order.
func (s *Filesystem) List(
	ctx context.Context,
	path string,
	options filesystem.ListOptions,
) (filesystem.ListPage, error) {
	options = options.Normalize()
	if err := ctx.Err(); err != nil {
		return filesystem.ListPage{}, err
	}
	if err := filesystem.ValidatePath(path); err != nil {
		return filesystem.ListPage{}, err
	}
	root, err := s.openRoot("list", path)
	if err != nil {
		return filesystem.ListPage{}, err
	}
	defer closeRoot(root)

	entries, err := listEntries(ctx, root.FS(), path, options.Recursive)
	if err != nil {
		return filesystem.ListPage{}, wrapPathError("list", path, err)
	}
	entries = filterEntries(entries, options.Kind)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})

	return paginate(entries, options), nil
}

// Move moves src to dst atomically when the local operating system permits it.
func (s *Filesystem) Move(ctx context.Context, src, dst string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := filesystem.ValidateFilePath(src); err != nil {
		return err
	}
	if err := filesystem.ValidateFilePath(dst); err != nil {
		return err
	}
	root, err := s.openRoot("move", src)
	if err != nil {
		return err
	}
	defer closeRoot(root)

	if err := root.Rename(nativePath(src), nativePath(dst)); err != nil {
		return wrapPathError("move", src, err)
	}
	return nil
}

// Link creates a hard link from src to dst.
func (s *Filesystem) Link(ctx context.Context, src, dst string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := filesystem.ValidateFilePath(src); err != nil {
		return err
	}
	if err := filesystem.ValidateFilePath(dst); err != nil {
		return err
	}
	root, err := s.openRoot("link", src)
	if err != nil {
		return err
	}
	defer closeRoot(root)

	if err := root.Link(nativePath(src), nativePath(dst)); err != nil {
		return wrapPathError("link", src, err)
	}
	return nil
}

// Symlink creates a relative symbolic link. Access through this filesystem
// rejects links whose resolution escapes the configured root.
func (s *Filesystem) Symlink(ctx context.Context, target, link string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := filesystem.ValidateFilePath(target); err != nil {
		return err
	}
	if err := filesystem.ValidateFilePath(link); err != nil {
		return err
	}
	root, err := s.openRoot("symlink", link)
	if err != nil {
		return err
	}
	defer closeRoot(root)

	targetPath := nativePath(target)
	linkPath := nativePath(link)
	relativeTarget, err := filepath.Rel(filepath.Dir(linkPath), targetPath)
	if err != nil {
		return wrapPathError("symlink", target, err)
	}
	if err := root.Symlink(relativeTarget, linkPath); err != nil {
		return wrapPathError("symlink", link, err)
	}
	return nil
}

// MakeDirectory creates path and any missing parent directories.
func (s *Filesystem) MakeDirectory(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := filesystem.ValidatePath(path); err != nil {
		return err
	}
	root, err := s.openRoot("mkdir", path)
	if err != nil {
		return err
	}
	defer closeRoot(root)

	if err := root.MkdirAll(nativePath(path), 0o755); err != nil {
		return wrapPathError("mkdir", path, err)
	}
	return nil
}

// DeleteDirectory recursively removes path. The logical root cannot be removed.
func (s *Filesystem) DeleteDirectory(ctx context.Context, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if path == "." {
		return &fs.PathError{Op: "remove", Path: path, Err: filesystem.ErrInvalidPath}
	}
	if err := filesystem.ValidatePath(path); err != nil {
		return err
	}
	root, err := s.openRoot("remove", path)
	if err != nil {
		return err
	}
	defer closeRoot(root)

	if err := root.RemoveAll(nativePath(path)); err != nil {
		return wrapPathError("remove", path, err)
	}
	return nil
}

func (s *Filesystem) openRoot(op, path string) (*os.Root, error) {
	root, err := os.OpenRoot(s.root)
	if err != nil {
		return nil, wrapPathError(op, path, err)
	}
	return root, nil
}

func nativePath(path string) string {
	return filepath.FromSlash(path)
}

func closeRoot(root *os.Root) {
	_ = root.Close()
}

func listEntries(
	ctx context.Context,
	storage fs.FS,
	root string,
	recursive bool,
) ([]filesystem.Entry, error) {
	info, err := fs.Stat(storage, root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, nil
	}

	if !recursive {
		dirEntries, err := fs.ReadDir(storage, root)
		if err != nil {
			return nil, err
		}

		entries := make([]filesystem.Entry, 0, len(dirEntries))
		for _, dirEntry := range dirEntries {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			entry, err := entryFromDirEntry(logicalJoin(root, dirEntry.Name()), dirEntry)
			if err != nil {
				return nil, err
			}
			if entry.IsDir() {
				continue
			}
			entries = append(entries, entry)
		}
		return entries, nil
	}

	var entries []filesystem.Entry
	err = fs.WalkDir(storage, root, func(path string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if path == root {
			return nil
		}
		if dirEntry.IsDir() {
			return nil
		}
		entry, err := entryFromDirEntry(path, dirEntry)
		if err != nil {
			return err
		}
		entries = append(entries, entry)
		return nil
	})
	return entries, err
}

func logicalJoin(directory, name string) string {
	if directory == "." {
		return name
	}
	return pathpkg.Join(directory, name)
}

func entryFromDirEntry(path string, dirEntry fs.DirEntry) (filesystem.Entry, error) {
	info, err := dirEntry.Info()
	if err != nil {
		return filesystem.Entry{}, err
	}
	return entryFromInfo(path, info), nil
}

func entryFromInfo(path string, info fs.FileInfo) filesystem.Entry {
	kind := filesystem.EntryKindFile
	if info.IsDir() {
		kind = filesystem.EntryKindDirectory
	}
	return filesystem.Entry{
		Path:         path,
		Kind:         kind,
		Size:         info.Size(),
		LastModified: info.ModTime(),
	}
}

func filterEntries(entries []filesystem.Entry, kind filesystem.EntryKind) []filesystem.Entry {
	if kind == filesystem.EntryKindAny {
		return entries
	}

	filtered := entries[:0]
	for _, entry := range entries {
		if entry.Kind == kind {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func paginate(entries []filesystem.Entry, options filesystem.ListOptions) filesystem.ListPage {
	start := sort.Search(len(entries), func(i int) bool {
		return entries[i].Path > options.Cursor
	})
	end := len(entries)
	if options.Limit > 0 && start+options.Limit < end {
		end = start + options.Limit
	}

	page := filesystem.ListPage{Entries: entries[start:end]}
	if end < len(entries) && end > start {
		page.NextCursor = entries[end-1].Path
	}
	return page
}

func wrapPathError(op, path string, err error) error {
	if errors.Is(err, os.ErrNotExist) {
		err = filesystem.ErrNotFound
	}
	return &fs.PathError{Op: op, Path: path, Err: err}
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(buffer)
}
