package local

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-fries/fries/filesystem/v3"
)

type Filesystem struct {
	root     string
	prefixer *filesystem.PathPrefixer
}

var _ filesystem.Filesystem = (*Filesystem)(nil)

func NewStorage(root string) *Filesystem {
	return &Filesystem{
		root:     root,
		prefixer: filesystem.NewPathPrefixer(root),
	}
}

func (s *Filesystem) Read(_ context.Context, path string) ([]byte, error) {
	return os.ReadFile(s.prefixer.Prefix(path))
}

func (s *Filesystem) Write(_ context.Context, path string, value []byte) error {
	return os.WriteFile(s.prefixer.Prefix(path), value, 0o644) //nolint:mnd
}

func (s *Filesystem) Delete(_ context.Context, path string) error {
	return os.Remove(s.prefixer.Prefix(path))
}

func (s *Filesystem) Exists(_ context.Context, path string) (bool, error) {
	_, err := os.Stat(s.prefixer.Prefix(path))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (s *Filesystem) Rename(_ context.Context, oldPath, newPath string) error {
	return os.Rename(s.prefixer.Prefix(oldPath), s.prefixer.Prefix(newPath))
}

func (s *Filesystem) Link(_ context.Context, oldPath, newPath string) error {
	return os.Link(s.prefixer.Prefix(oldPath), s.prefixer.Prefix(newPath))
}

func (s *Filesystem) Symlink(_ context.Context, oldPath, newPath string) error {
	_, _ = oldPath, newPath
	panic("not implemented") // TODO: Implement
	// return os.Symlink(s.prefixer.Prefix(oldPath), s.prefixer.Prefix(newPath))
}

func (s *Filesystem) Files(_ context.Context, path string) ([]string, error) {
	f, err := os.ReadDir(s.prefixer.Prefix(path))
	if err != nil {
		return nil, err
	}

	var files []string
	for _, file := range f {
		if !file.IsDir() {
			files = append(files, file.Name())
		}
	}
	return files, nil
}

func (s *Filesystem) AllFiles(ctx context.Context, path string) ([]string, error) {
	f, err := os.ReadDir(s.prefixer.Prefix(path))
	if err != nil {
		return nil, err
	}

	var files []string //molint:prealloc
	for _, file := range f {
		if !file.IsDir() {
			files = append(files, file.Name())
		} else {
			subFiles, err := s.AllFiles(ctx, file.Name())
			if err != nil {
				return nil, err
			}
			files = append(files, subFiles...)
		}
	}

	return files, nil
}

func (s *Filesystem) Directories(_ context.Context, path string) ([]string, error) {
	f, err := os.ReadDir(s.prefixer.Prefix(path))
	if err != nil {
		return nil, err
	}

	var dirs []string //nolint:prealloc
	for _, file := range f {
		if !file.IsDir() {
			continue
		}
		dirs = append(dirs, file.Name())
	}
	return dirs, nil
}

func (s *Filesystem) AllDirectories(ctx context.Context, path string) ([]string, error) {
	f, err := os.ReadDir(s.prefixer.Prefix(path))
	if err != nil {
		return nil, err
	}

	var dirs []string
	for _, file := range f {
		if !file.IsDir() {
			continue
		}
		subDirs, err := s.AllDirectories(ctx, file.Name())
		if err != nil {
			return nil, err
		}
		dirs = append(dirs, subDirs...)
	}
	return dirs, nil
}

func (s *Filesystem) MakeDirectory(_ context.Context, path string) error {
	return os.MkdirAll(s.prefixer.Prefix(path), 0o755) //nolint:mnd
}

func (s *Filesystem) DeleteDirectory(_ context.Context, path string) error {
	return os.RemoveAll(s.prefixer.Prefix(path))
}

func (s *Filesystem) IsFile(_ context.Context, path string) (bool, error) {
	info, err := os.Stat(s.prefixer.Prefix(path))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return !info.IsDir(), nil
}

func (s *Filesystem) IsDir(_ context.Context, path string) (bool, error) {
	info, err := os.Stat(s.prefixer.Prefix(path))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return info.IsDir(), nil
}

func (s *Filesystem) Size(_ context.Context, path string) (int64, error) {
	info, err := os.Stat(s.prefixer.Prefix(path))
	if err != nil {
		return 0, err
	}

	return info.Size(), nil
}

func (s *Filesystem) LastModified(_ context.Context, path string) (*time.Time, error) {
	info, err := os.Stat(s.prefixer.Prefix(path))
	if err != nil {
		return nil, err
	}

	return ptr(info.ModTime()), nil
}

func (s *Filesystem) Path(_ context.Context, path string) string {
	return s.prefixer.Prefix(path)
}

func (s *Filesystem) Name(_ context.Context, path string) string {
	path = s.prefixer.Prefix(path)
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func (s *Filesystem) Basename(_ context.Context, path string) string {
	return filepath.Base(s.prefixer.Prefix(path))
}

func (s *Filesystem) Dirname(_ context.Context, path string) string {
	return filepath.Dir(s.prefixer.Prefix(path))
}

func (s *Filesystem) Extension(_ context.Context, path string) string {
	return filepath.Ext(s.prefixer.Prefix(path))
}

func ptr[T any](v T) *T {
	return &v
}
