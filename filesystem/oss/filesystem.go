package oss

import (
	"context"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/go-fries/fries/filesystem/v3"
)

type Filesystem struct {
	client   *oss.Client
	bucket   string
	root     string
	prefixer *filesystem.PathPrefixer
}

type Option func(*Filesystem)

func WithRoot(root string) Option {
	return func(f *Filesystem) {
		f.root = root
	}
}

func New(client *oss.Client, bucket string, opts ...Option) *Filesystem {
	f := &Filesystem{
		client: client,
		bucket: bucket,
	}
	for _, opt := range opts {
		opt(f)
	}
	f.prefixer = filesystem.NewPathPrefixer(f.root)
	return f
}

func (f *Filesystem) Read(ctx context.Context, path string) ([]byte, error) {
	// TODO implement me
	panic("implement me")
}

func (f *Filesystem) Write(ctx context.Context, path string, value []byte) error {
	// TODO implement me
	panic("implement me")
}

func (f *Filesystem) Delete(ctx context.Context, path string) error {
	// TODO implement me
	panic("implement me")
}

func (f *Filesystem) Exists(ctx context.Context, path string) (bool, error) {
	// TODO implement me
	panic("implement me")
}

func (f *Filesystem) Rename(ctx context.Context, oldPath, newPath string) error {
	// TODO implement me
	panic("implement me")
}

func (f *Filesystem) Link(ctx context.Context, oldPath, newPath string) error {
	// TODO implement me
	panic("implement me")
}

func (f *Filesystem) Symlink(ctx context.Context, oldPath, newPath string) error {
	// TODO implement me
	panic("implement me")
}

func (f *Filesystem) Files(ctx context.Context, path string) ([]string, error) {
	// TODO implement me
	panic("implement me")
}

func (f *Filesystem) AllFiles(ctx context.Context, path string) ([]string, error) {
	// TODO implement me
	panic("implement me")
}

func (f *Filesystem) Directories(ctx context.Context, path string) ([]string, error) {
	// TODO implement me
	panic("implement me")
}

func (f *Filesystem) AllDirectories(ctx context.Context, path string) ([]string, error) {
	// TODO implement me
	panic("implement me")
}

func (f *Filesystem) MakeDirectory(ctx context.Context, path string) error {
	// TODO implement me
	panic("implement me")
}

func (f *Filesystem) DeleteDirectory(ctx context.Context, path string) error {
	// TODO implement me
	panic("implement me")
}

func (f *Filesystem) IsFile(ctx context.Context, path string) (bool, error) {
	// TODO implement me
	panic("implement me")
}

func (f *Filesystem) IsDir(ctx context.Context, path string) (bool, error) {
	// TODO implement me
	panic("implement me")
}

func (f *Filesystem) Size(ctx context.Context, path string) (int64, error) {
	// TODO implement me
	panic("implement me")
}

func (f *Filesystem) LastModified(ctx context.Context, path string) (*time.Time, error) {
	// TODO implement me
	panic("implement me")
}

func (f *Filesystem) Path(ctx context.Context, path string) string {
	// TODO implement me
	panic("implement me")
}

func (f *Filesystem) Name(ctx context.Context, path string) string {
	// TODO implement me
	panic("implement me")
}

func (f *Filesystem) Basename(ctx context.Context, path string) string {
	// TODO implement me
	panic("implement me")
}

func (f *Filesystem) Dirname(ctx context.Context, path string) string {
	// TODO implement me
	panic("implement me")
}

func (f *Filesystem) Extension(ctx context.Context, path string) string {
	// TODO implement me
	panic("implement me")
}

var _ filesystem.Filesystem = (*Filesystem)(nil)
