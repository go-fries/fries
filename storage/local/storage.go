package local

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-fries/fries/storage/v3"
)

type Storage struct {
	root     string
	prefixer *storage.PathPrefixer
}

var (
	_ storage.Storage = (*Storage)(nil)
)

func NewStorage(root string) *Storage {
	return &Storage{
		root:     root,
		prefixer: storage.NewPathPrefixer(root),
	}
}

func (s *Storage) Read(_ context.Context, path string) ([]byte, error) {
	return os.ReadFile(s.prefixer.Prefix(path))
}

func (s *Storage) Write(_ context.Context, path string, value []byte) error {
	return os.WriteFile(s.prefixer.Prefix(path), value, 0o644)
}

func (s *Storage) Delete(_ context.Context, path string) error {
	return os.Remove(s.prefixer.Prefix(path))
}

func (s *Storage) Exists(_ context.Context, path string) (bool, error) {
	_, err := os.Stat(s.prefixer.Prefix(path))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (s *Storage) Rename(_ context.Context, oldPath, newPath string) error {
	return os.Rename(s.prefixer.Prefix(oldPath), s.prefixer.Prefix(newPath))
}

func (s *Storage) Link(_ context.Context, oldPath, newPath string) error {
	return os.Link(s.prefixer.Prefix(oldPath), s.prefixer.Prefix(newPath))
}

func (s *Storage) Symlink(_ context.Context, oldPath, newPath string) error {
	return os.Symlink(s.prefixer.Prefix(oldPath), s.prefixer.Prefix(newPath))
}

func (s *Storage) Files(_ context.Context, path string) ([]string, error) {
	f, err := os.ReadDir(s.prefixer.Prefix(path))
	if err != nil {
		return nil, err
	}

	var files []string //nolint:prealloc
	for _, file := range f {
		if !file.IsDir() {
			files = append(files, file.Name())
		}
	}
	return files, nil
}

func (s *Storage) AllFiles(ctx context.Context, path string) ([]string, error) {
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

func (s *Storage) Directories(_ context.Context, path string) ([]string, error) {
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

func (s *Storage) AllDirectories(ctx context.Context, path string) ([]string, error) {
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

func (s *Storage) MakeDirectory(_ context.Context, path string) error {
	return os.MkdirAll(s.prefixer.Prefix(path), 0o755) //nolint:mnd
}

func (s *Storage) DeleteDirectory(_ context.Context, path string) error {
	return os.RemoveAll(s.prefixer.Prefix(path))
}

func (s *Storage) IsFile(_ context.Context, path string) (bool, error) {
	info, err := os.Stat(s.prefixer.Prefix(path))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return !info.IsDir(), nil
}

func (s *Storage) IsDir(_ context.Context, path string) (bool, error) {
	info, err := os.Stat(s.prefixer.Prefix(path))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return info.IsDir(), nil
}

func (s *Storage) Size(_ context.Context, path string) (int64, error) {
	info, err := os.Stat(s.prefixer.Prefix(path))
	if err != nil {
		return 0, err
	}

	return info.Size(), nil
}

func (s *Storage) LastModified(_ context.Context, path string) (*time.Time, error) {
	info, err := os.Stat(s.prefixer.Prefix(path))
	if err != nil {
		return nil, err
	}

	return ptr(info.ModTime()), nil
}

func (s *Storage) Path(_ context.Context, path string) string {
	return s.prefixer.Prefix(path)
}

func (s *Storage) Name(_ context.Context, path string) string {
	path = s.prefixer.Prefix(path)
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func (s *Storage) Basename(_ context.Context, path string) string {
	return filepath.Base(s.prefixer.Prefix(path))
}

func (s *Storage) Dirname(_ context.Context, path string) string {
	return filepath.Dir(s.prefixer.Prefix(path))
}

func (s *Storage) Extension(_ context.Context, path string) string {
	return filepath.Ext(s.prefixer.Prefix(path))
}

func ptr[T any](v T) *T {
	return &v
}
