package s3

import (
	"bytes"
	"context"
	"errors"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/go-fries/fries/storage/v3"
)

type Storage struct {
	s3       *s3.Client
	prefixer *storage.PathPrefixer

	root   string
	bucket string
}

var (
	_ storage.Storage  = (*Storage)(nil)
	_ storage.Copyable = (*Storage)(nil)
)

type Option func(*Storage)

func WithRoot(root string) Option {
	return func(s *Storage) {
		s.root = root
	}
}

func New(s3 *s3.Client, bucket string, opts ...Option) *Storage {
	s := &Storage{
		s3:     s3,
		bucket: bucket,
		root:   "",
	}
	for _, opt := range opts {
		opt(s)
	}

	s.prefixer = storage.NewPathPrefixer(s.root)

	return s
}

func (s *Storage) Read(ctx context.Context, path string) ([]byte, error) {
	output, err := s.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: ptr(s.bucket),
		Key:    ptr(s.prefixer.Prefix(path)),
	})
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func (s *Storage) Write(ctx context.Context, path string, value []byte) error {
	_, err := s.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: ptr(s.bucket),
		Key:    ptr(s.prefixer.Prefix(path)),
		Body:   io.NopCloser(bytes.NewBuffer(value)),
	})
	return err
}

func (s *Storage) Exists(ctx context.Context, path string) (bool, error) {
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

func (s *Storage) Rename(ctx context.Context, oldPath, newPath string) error {
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

func (s *Storage) MakeDirectory(ctx context.Context, path string) error {
	_, err := s.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: ptr(s.bucket),
		Key:    ptr(s.prefixer.Prefix(path) + "/"),
		Body:   io.NopCloser(bytes.NewBuffer([]byte{})),
	})
	return err
}

func (s *Storage) DeleteDirectory(ctx context.Context, path string) error {
	_, err := s.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: ptr(s.bucket),
		Key:    ptr(s.prefixer.Prefix(path) + "/"),
	})
	return err
}

func (s *Storage) Size(ctx context.Context, path string) (int64, error) {
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

func (s *Storage) LastModified(ctx context.Context, path string) (*time.Time, error) {
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

func (s *Storage) Path(_ context.Context, path string) string {
	return s.prefixer.Prefix(path)
}

func (s *Storage) Name(ctx context.Context, path string) string {

}

func (s *Storage) Basename(ctx context.Context, path string) string {
	// TODO implement me
	panic("implement me")
}

func (s *Storage) Dirname(ctx context.Context, path string) string {
	// TODO implement me
	panic("implement me")
}

func (s *Storage) Extension(ctx context.Context, path string) string {
	// TODO implement me
	panic("implement me")
}

func (s *Storage) Delete(ctx context.Context, path string) error {
	_, err := s.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: ptr(s.bucket),
		Key:    ptr(s.prefixer.Prefix(path)),
	})
	return err
}

//
// func (s *Storage) Move(ctx context.Context, oldPath, newPath string) error {
// 	if err := s.Copy(ctx, oldPath, newPath); err != nil {
// 		return err
// 	}
// 	return s.Delete(ctx, oldPath)
// }
//
// func (s *Storage) Copy(ctx context.Context, oldPath, newPath string) error {
// 	_, err := s.s3.CopyObject(ctx, &s3.CopyObjectInput{
// 		Bucket:     ptr(s.bucket),
// 		CopySource: ptr(s.bucket + "/" + s.prefixer.Prefix(oldPath)),
// 		Key:        ptr(s.prefixer.Prefix(newPath)),
// 	})
// 	return err
// }
//
// func (s *Storage) Link(context.Context, string, string) error {
// 	return storage.ErrNotSupported
// }
//
// func (s *Storage) Symlink(context.Context, string, string) error {
// 	return storage.ErrNotSupported
// }
//
// func (s *Storage) Files(ctx context.Context, path string) ([]string, error) {
// 	output, err := s.s3.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
// 		Bucket: ptr(s.bucket),
// 		Prefix: ptr(s.prefixer.Prefix(path)),
// 	})
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	var files []string
// 	for _, obj := range output.Contents {
// 		files = append(files, *obj.Key)
// 	}
// 	return files, nil
// }
//
// func (s *Storage) AllFiles(ctx context.Context, path string) ([]string, error) {
// 	output, err := s.s3.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
// 		Bucket: ptr(s.bucket),
// 		Prefix: ptr(s.prefixer.Prefix(path)),
// 	})
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	var files []string
// 	for _, obj := range output.Contents {
// 		files = append(files, *obj.Key)
// 	}
// 	return files, nil
// }
//
// func (s *Storage) Directories(ctx context.Context, path string) ([]string, error) {
// 	output, err := s.s3.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
// 		Bucket: ptr(s.bucket),
// 		Prefix: ptr(s.prefixer.Prefix(path)),
// 	})
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	var dirs []string
// 	for _, obj := range output.Contents {
// 		dirs = append(dirs, *obj.Key)
// 	}
// 	return dirs, nil
// }
//
// func (s *Storage) AllDirectories(ctx context.Context, path string) ([]string, error) {
// 	output, err := s.s3.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
// 		Bucket: ptr(s.bucket),
// 		Prefix: ptr(s.prefixer.Prefix(path)),
// 	})
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	var dirs []string
// 	for _, obj := range output.Contents {
// 		dirs = append(dirs, *obj.Key)
// 	}
// 	return dirs, nil
// }
//
// func (s *Storage) IsFile(ctx context.Context, path string) (bool, error) {
// 	// TODO implement me
// 	panic("implement me")
// }
//
// func (s *Storage) IsDir(ctx context.Context, path string) (bool, error) {
// 	// TODO implement me
// 	panic("implement me")
// }

func ptr[T any](v T) *T {
	return &v
}
