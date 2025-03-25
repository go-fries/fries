package s3

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/go-fries/fries/filesystem/v3"
)

type Filesystem struct {
	s3       *s3.Client
	prefixer *filesystem.PathPrefixer

	root   string
	bucket string
}

var (
	_ filesystem.Filesystem = (*Filesystem)(nil)
	_ filesystem.Copyable   = (*Filesystem)(nil)
)

type Option func(*Filesystem)

func WithRoot(root string) Option {
	return func(s *Filesystem) {
		s.root = root
	}
}

func New(s3 *s3.Client, bucket string, opts ...Option) *Filesystem {
	s := &Filesystem{
		s3:     s3,
		bucket: bucket,
		root:   "",
	}
	for _, opt := range opts {
		opt(s)
	}

	s.prefixer = filesystem.NewPathPrefixer(s.root)

	return s
}

func (s *Filesystem) Read(ctx context.Context, path string) ([]byte, error) {
	output, err := s.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: ptr(s.bucket),
		Key:    ptr(s.prefixer.Prefix(path)),
	})
	if err != nil {
		return nil, err
	}
	defer output.Body.Close() // nolint:errcheck

	body, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (s *Filesystem) Write(ctx context.Context, path string, value []byte) error {
	_, err := s.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        ptr(s.bucket),
		Key:           ptr(s.prefixer.Prefix(path)),
		Body:          io.NopCloser(bytes.NewBuffer(value)),
		ContentType:   ptr(http.DetectContentType(value)),
		ContentLength: ptr(int64(len(value))),
	})
	return err
}

func (s *Filesystem) Exists(ctx context.Context, path string) (bool, error) {
	_, err := s.s3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: ptr(s.bucket),
		Key:    ptr(s.prefixer.Prefix(path)),
	})
	if err != nil {
		var nf *types.NotFound
		if errors.As(err, &nf) {
			return false, nil
		}
		// Alternative approach using response error
		var responseError smithy.APIError
		if errors.As(err, &responseError) {
			if responseError.ErrorCode() == "NotFound" {
				return false, nil
			}
		}
		return false, err
	}

	return true, nil
}

func (s *Filesystem) Rename(ctx context.Context, oldPath, newPath string) error {
	_, err := s.s3.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     ptr(s.bucket),
		CopySource: ptr(s.bucket + "/" + s.prefixer.Prefix(oldPath)),
		Key:        ptr(s.prefixer.Prefix(newPath)),
	})
	if err != nil {
		return err
	}

	return s.Delete(ctx, oldPath)
}

func (s *Filesystem) MakeDirectory(ctx context.Context, path string) error {
	_, err := s.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: ptr(s.bucket),
		Key:    ptr(s.prefixer.Prefix(path) + "/"),
		Body:   io.NopCloser(bytes.NewBuffer([]byte{})),
	})
	return err
}

func (s *Filesystem) DeleteDirectory(ctx context.Context, path string) error {
	_, err := s.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: ptr(s.bucket),
		Key:    ptr(s.prefixer.Prefix(path) + "/"),
	})
	return err
}

func (s *Filesystem) Size(ctx context.Context, path string) (int64, error) {
	output, err := s.s3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: ptr(s.bucket),
		Key:    ptr(s.prefixer.Prefix(path)),
	})
	if err != nil {
		return 0, err
	}

	if output.ContentLength == nil {
		return 0, errors.New("storage: content length is nil")
	}

	return *output.ContentLength, nil
}

func (s *Filesystem) LastModified(ctx context.Context, path string) (*time.Time, error) {
	output, err := s.s3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: ptr(s.bucket),
		Key:    ptr(s.prefixer.Prefix(path)),
	})
	if err != nil {
		return nil, err
	}

	if output.LastModified == nil {
		return nil, errors.New("storage: last modified is nil")
	}

	return output.LastModified, nil
}

func (s *Filesystem) Path(_ context.Context, path string) string {
	return s.prefixer.Prefix(path)
}

func (s *Filesystem) Name(ctx context.Context, path string) string { //nolint:revive // todo: remove nolint:revive
	// TODO implement me
	panic("implement me")
}

func (s *Filesystem) Basename(ctx context.Context, path string) string { //nolint:revive // todo: remove nolint:revive
	// TODO implement me
	panic("implement me")
}

func (s *Filesystem) Dirname(ctx context.Context, path string) string { //nolint:revive // todo: remove nolint:revive
	// TODO implement me
	panic("implement me")
}

func (s *Filesystem) Extension(ctx context.Context, path string) string { //nolint:revive // todo: remove nolint:revive
	// TODO implement me
	panic("implement me")
}

func (s *Filesystem) Delete(ctx context.Context, path string) error {
	_, err := s.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: ptr(s.bucket),
		Key:    ptr(s.prefixer.Prefix(path)),
	})
	return err
}

func (s *Filesystem) Link(context.Context, string, string) error {
	return filesystem.ErrNotSupported
}

func (s *Filesystem) Symlink(context.Context, string, string) error {
	return filesystem.ErrNotSupported
}

func (s *Filesystem) Files(ctx context.Context, path string) ([]string, error) { //nolint:revive // todo: remove nolint:revive
	// TODO implement me
	panic("implement me")
}

func (s *Filesystem) AllFiles(ctx context.Context, path string) ([]string, error) { //nolint:revive // todo: remove nolint:revive
	// TODO implement me
	panic("implement me")
}

func (s *Filesystem) Directories(ctx context.Context, path string) ([]string, error) { //nolint:revive // todo: remove nolint:revive
	// TODO implement me
	panic("implement me")
}

func (s *Filesystem) AllDirectories(ctx context.Context, path string) ([]string, error) { //nolint:revive // todo: remove nolint:revive
	// TODO implement me
	panic("implement me")
}

func (s *Filesystem) IsFile(ctx context.Context, path string) (bool, error) { //nolint:revive // todo: remove nolint:revive
	// TODO implement me
	panic("implement me")
}

func (s *Filesystem) IsDir(ctx context.Context, path string) (bool, error) { //nolint:revive // todo: remove nolint:revive
	// TODO implement me
	panic("implement me")
}

func (s *Filesystem) Copy(ctx context.Context, oldPath, newPath string) error { //nolint:revive // todo: remove nolint:revive
	// TODO implement me
	panic("implement me")
}

func ptr[T any](v T) *T {
	return &v
}
