package s3

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"maps"
	"net/url"
	"sort"
	"strings"

	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/go-fries/fries/filesystem/v4"
)

type s3Client interface {
	GetObject(
		context.Context,
		*awss3.GetObjectInput,
		...func(*awss3.Options),
	) (*awss3.GetObjectOutput, error)
	PutObject(
		context.Context,
		*awss3.PutObjectInput,
		...func(*awss3.Options),
	) (*awss3.PutObjectOutput, error)
	DeleteObject(
		context.Context,
		*awss3.DeleteObjectInput,
		...func(*awss3.Options),
	) (*awss3.DeleteObjectOutput, error)
	HeadObject(
		context.Context,
		*awss3.HeadObjectInput,
		...func(*awss3.Options),
	) (*awss3.HeadObjectOutput, error)
	ListObjectsV2(
		context.Context,
		*awss3.ListObjectsV2Input,
		...func(*awss3.Options),
	) (*awss3.ListObjectsV2Output, error)
	CopyObject(
		context.Context,
		*awss3.CopyObjectInput,
		...func(*awss3.Options),
	) (*awss3.CopyObjectOutput, error)
}

// Filesystem stores logical filesystem paths in an Amazon S3 bucket.
type Filesystem struct {
	client   s3Client
	prefixer *filesystem.PathPrefixer
	bucket   string
}

var (
	_ filesystem.Driver = (*Filesystem)(nil)
	_ filesystem.Copier = (*Filesystem)(nil)
	_ filesystem.Mover  = (*Filesystem)(nil)
)

// New creates an S3-backed filesystem.
func New(client *awss3.Client, bucket string, opts ...Option) *Filesystem {
	return newFilesystem(client, bucket, opts...)
}

func newFilesystem(client s3Client, bucket string, opts ...Option) *Filesystem {
	cfg := newConfig(opts...)
	return &Filesystem{
		client:   client,
		bucket:   bucket,
		prefixer: filesystem.NewPathPrefixer(cfg.root),
	}
}

// Open opens path for streaming reads.
func (s *Filesystem) Open(ctx context.Context, path string) (io.ReadCloser, error) {
	if err := filesystem.ValidateFilePath(path); err != nil {
		return nil, err
	}
	output, err := s.client.GetObject(ctx, &awss3.GetObjectInput{
		Bucket: ptr(s.bucket),
		Key:    ptr(s.prefixer.Prefix(path)),
	})
	if err != nil {
		return nil, wrapPathError("open", path, err)
	}
	return output.Body, nil
}

// Put writes src to path, replacing an existing object.
func (s *Filesystem) Put(
	ctx context.Context,
	path string,
	src io.Reader,
	options filesystem.PutOptions,
) error {
	if err := filesystem.ValidateFilePath(path); err != nil {
		return err
	}
	input := &awss3.PutObjectInput{
		Bucket:   ptr(s.bucket),
		Key:      ptr(s.prefixer.Prefix(path)),
		Body:     src,
		Metadata: cloneMetadata(options.Metadata),
	}
	if options.ContentType != "" {
		input.ContentType = ptr(options.ContentType)
	}
	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		return wrapPathError("put", path, err)
	}
	return nil
}

// Delete removes an object.
func (s *Filesystem) Delete(ctx context.Context, path string) error {
	if err := filesystem.ValidateFilePath(path); err != nil {
		return err
	}
	_, err := s.client.DeleteObject(ctx, &awss3.DeleteObjectInput{
		Bucket: ptr(s.bucket),
		Key:    ptr(s.prefixer.Prefix(path)),
	})
	if err != nil {
		return wrapPathError("delete", path, err)
	}
	return nil
}

// Stat returns object metadata.
func (s *Filesystem) Stat(ctx context.Context, path string) (filesystem.Entry, error) {
	if err := filesystem.ValidatePath(path); err != nil {
		return filesystem.Entry{}, err
	}
	if err := ctx.Err(); err != nil {
		return filesystem.Entry{}, err
	}
	if path == "." {
		return filesystem.Entry{Path: ".", Kind: filesystem.EntryKindDirectory}, nil
	}
	output, err := s.client.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: ptr(s.bucket),
		Key:    ptr(s.prefixer.Prefix(path)),
	})
	if err != nil {
		return filesystem.Entry{}, wrapPathError("stat", path, err)
	}

	entry := filesystem.Entry{
		Path:        path,
		Kind:        filesystem.EntryKindFile,
		ContentType: dereference(output.ContentType),
		Metadata:    cloneMetadata(output.Metadata),
	}
	if output.ContentLength != nil {
		entry.Size = *output.ContentLength
	}
	if output.LastModified != nil {
		entry.LastModified = *output.LastModified
	}
	return entry, nil
}

// List returns one page of objects and virtual directories below path.
func (s *Filesystem) List(
	ctx context.Context,
	path string,
	options filesystem.ListOptions,
) (filesystem.ListPage, error) {
	if err := filesystem.ValidatePath(path); err != nil {
		return filesystem.ListPage{}, err
	}
	input := &awss3.ListObjectsV2Input{
		Bucket: ptr(s.bucket),
		Prefix: ptr(directoryPrefix(s.prefixer.Prefix(path))),
	}
	if !options.Recursive {
		input.Delimiter = ptr("/")
	}
	if options.Cursor != "" {
		input.ContinuationToken = ptr(options.Cursor)
	}
	if options.Limit > 0 {
		limit := min(options.Limit, 1000)
		input.MaxKeys = ptr(int32(limit))
	}

	output, err := s.client.ListObjectsV2(ctx, input)
	if err != nil {
		return filesystem.ListPage{}, wrapPathError("list", path, err)
	}

	page := filesystem.ListPage{Entries: s.entries(output, options.Kind)}
	if output.NextContinuationToken != nil {
		page.NextCursor = *output.NextContinuationToken
	}
	return page, nil
}

// Copy copies src to dst using S3's native server-side copy operation.
func (s *Filesystem) Copy(ctx context.Context, src, dst string) error {
	if err := filesystem.ValidateFilePath(src); err != nil {
		return err
	}
	if err := filesystem.ValidateFilePath(dst); err != nil {
		return err
	}
	copySource := url.PathEscape(s.bucket + "/" + s.prefixer.Prefix(src))
	_, err := s.client.CopyObject(ctx, &awss3.CopyObjectInput{
		Bucket:     ptr(s.bucket),
		CopySource: ptr(copySource),
		Key:        ptr(s.prefixer.Prefix(dst)),
	})
	if err != nil {
		return wrapPathError("copy", src, err)
	}
	return nil
}

// Move copies src to dst and then deletes src. The operation is not atomic.
func (s *Filesystem) Move(ctx context.Context, src, dst string) error {
	if err := s.Copy(ctx, src, dst); err != nil {
		return err
	}
	return s.Delete(ctx, src)
}

func (s *Filesystem) entries(
	output *awss3.ListObjectsV2Output,
	kind filesystem.EntryKind,
) []filesystem.Entry {
	byPath := make(map[string]filesystem.Entry, len(output.Contents)+len(output.CommonPrefixes))
	for _, object := range output.Contents {
		key := dereference(object.Key)
		entryKind := filesystem.EntryKindFile
		if strings.HasSuffix(key, "/") {
			entryKind = filesystem.EntryKindDirectory
			key = strings.TrimSuffix(key, "/")
		}
		path, ok := s.prefixer.Strip(key)
		if !ok || path == "." || !filesystem.ValidPath(path) || !matchesKind(entryKind, kind) {
			continue
		}
		entry := filesystem.Entry{Path: path, Kind: entryKind}
		if object.Size != nil {
			entry.Size = *object.Size
		}
		if object.LastModified != nil {
			entry.LastModified = *object.LastModified
		}
		byPath[path] = entry
	}
	for _, commonPrefix := range output.CommonPrefixes {
		key := strings.TrimSuffix(dereference(commonPrefix.Prefix), "/")
		path, ok := s.prefixer.Strip(key)
		if !ok || path == "." || !filesystem.ValidPath(path) ||
			!matchesKind(filesystem.EntryKindDirectory, kind) {
			continue
		}
		byPath[path] = filesystem.Entry{Path: path, Kind: filesystem.EntryKindDirectory}
	}

	entries := make([]filesystem.Entry, 0, len(byPath))
	for _, entry := range byPath {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})
	return entries
}

func directoryPrefix(prefix string) string {
	if prefix == "" {
		return ""
	}
	return strings.TrimSuffix(prefix, "/") + "/"
}

func matchesKind(entryKind, filter filesystem.EntryKind) bool {
	return filter == filesystem.EntryKindAny || entryKind == filter
}

func wrapPathError(op, path string, err error) error {
	if isNotFound(err) {
		err = filesystem.ErrNotFound
	}
	return &fs.PathError{Op: op, Path: path, Err: err}
}

func isNotFound(err error) bool {
	var notFound *types.NotFound
	if errors.As(err, &notFound) {
		return true
	}
	var noSuchKey *types.NoSuchKey
	if errors.As(err, &noSuchKey) {
		return true
	}
	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		switch apiError.ErrorCode() {
		case "NotFound", "NoSuchKey", "404":
			return true
		}
	}
	return false
}

func cloneMetadata(metadata map[string]string) map[string]string {
	return maps.Clone(metadata)
}

func dereference(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func ptr[T any](value T) *T {
	return &value
}
