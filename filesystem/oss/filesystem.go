package oss

import (
	"bytes"
	"context"
	"io"
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
	fs := &Filesystem{
		client: client,
		bucket: bucket,
	}
	for _, opt := range opts {
		opt(fs)
	}
	fs.prefixer = filesystem.NewPathPrefixer(fs.root)
	return fs
}

func (fs *Filesystem) Read(ctx context.Context, path string) ([]byte, error) {
	result, err := fs.client.GetObject(ctx, &oss.GetObjectRequest{
		Bucket: oss.Ptr(fs.bucket),
		Key:    oss.Ptr(fs.prefixer.Prefix(path)),
	})
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()
	return io.ReadAll(result.Body)
}

func (fs *Filesystem) Write(ctx context.Context, path string, value []byte) error {
	_, err := fs.client.PutObject(ctx, &oss.PutObjectRequest{
		Bucket: oss.Ptr(fs.bucket),
		Key:    oss.Ptr(fs.prefixer.Prefix(path)),
		Body:   bytes.NewBuffer(value),
	})
	return err
}

func (fs *Filesystem) Delete(ctx context.Context, path string) error {
	_, err := fs.client.DeleteObject(ctx, &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(fs.bucket),
		Key:    oss.Ptr(fs.prefixer.Prefix(path)),
	})
	return err
}

func (fs *Filesystem) Exists(ctx context.Context, path string) (bool, error) {
	return fs.client.IsObjectExist(ctx, fs.bucket, fs.prefixer.Prefix(path))
}

func (fs *Filesystem) Rename(ctx context.Context, oldPath, newPath string) error {
	// TODO implement me
	panic("implement me")
}

func (fs *Filesystem) Link(ctx context.Context, oldPath, newPath string) error {
	// TODO implement me
	panic("implement me")
}

func (fs *Filesystem) Symlink(ctx context.Context, oldPath, newPath string) error {
	// TODO implement me
	panic("implement me")
}

func (fs *Filesystem) Files(ctx context.Context, path string) ([]string, error) {
	// TODO implement me
	panic("implement me")
}

func (fs *Filesystem) AllFiles(ctx context.Context, path string) ([]string, error) {
	// TODO implement me
	panic("implement me")
}

func (fs *Filesystem) Directories(ctx context.Context, path string) ([]string, error) {
	// TODO implement me
	panic("implement me")
}

func (fs *Filesystem) AllDirectories(ctx context.Context, path string) ([]string, error) {
	// TODO implement me
	panic("implement me")
}

func (fs *Filesystem) MakeDirectory(ctx context.Context, path string) error {
	// TODO implement me
	panic("implement me")
}

func (fs *Filesystem) DeleteDirectory(ctx context.Context, path string) error {
	// TODO implement me
	panic("implement me")
}

func (fs *Filesystem) IsFile(ctx context.Context, path string) (bool, error) {
	// TODO implement me
	panic("implement me")
}

func (fs *Filesystem) IsDir(ctx context.Context, path string) (bool, error) {
	// TODO implement me
	panic("implement me")
}

func (fs *Filesystem) Size(ctx context.Context, path string) (int64, error) {
	// TODO implement me
	panic("implement me")
}

func (fs *Filesystem) LastModified(ctx context.Context, path string) (*time.Time, error) {
	// TODO implement me
	panic("implement me")
}

func (fs *Filesystem) Path(ctx context.Context, path string) string {
	// TODO implement me
	panic("implement me")
}

func (fs *Filesystem) Name(ctx context.Context, path string) string {
	// TODO implement me
	panic("implement me")
}

func (fs *Filesystem) Basename(ctx context.Context, path string) string {
	// TODO implement me
	panic("implement me")
}

func (fs *Filesystem) Dirname(ctx context.Context, path string) string {
	// TODO implement me
	panic("implement me")
}

func (fs *Filesystem) Extension(ctx context.Context, path string) string {
	// TODO implement me
	panic("implement me")
}

var _ filesystem.Filesystem = (*Filesystem)(nil)
