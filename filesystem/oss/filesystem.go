package oss

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"maps"
	"sort"
	"strings"

	aliyunoss "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/go-fries/fries/filesystem/v4"
)

type ossClient interface {
	GetObject(
		context.Context,
		*aliyunoss.GetObjectRequest,
		...func(*aliyunoss.Options),
	) (*aliyunoss.GetObjectResult, error)
	PutObject(
		context.Context,
		*aliyunoss.PutObjectRequest,
		...func(*aliyunoss.Options),
	) (*aliyunoss.PutObjectResult, error)
	DeleteObject(
		context.Context,
		*aliyunoss.DeleteObjectRequest,
		...func(*aliyunoss.Options),
	) (*aliyunoss.DeleteObjectResult, error)
	HeadObject(
		context.Context,
		*aliyunoss.HeadObjectRequest,
		...func(*aliyunoss.Options),
	) (*aliyunoss.HeadObjectResult, error)
	ListObjectsV2(
		context.Context,
		*aliyunoss.ListObjectsV2Request,
		...func(*aliyunoss.Options),
	) (*aliyunoss.ListObjectsV2Result, error)
	CopyObject(
		context.Context,
		*aliyunoss.CopyObjectRequest,
		...func(*aliyunoss.Options),
	) (*aliyunoss.CopyObjectResult, error)
}

// Filesystem stores logical filesystem paths in an Alibaba Cloud OSS bucket.
type Filesystem struct {
	client   ossClient
	bucket   string
	prefixer *filesystem.PathPrefixer
}

var (
	_ filesystem.Driver = (*Filesystem)(nil)
	_ filesystem.Copier = (*Filesystem)(nil)
	_ filesystem.Mover  = (*Filesystem)(nil)
)

// New creates an OSS-backed filesystem.
func New(client *aliyunoss.Client, bucket string, opts ...Option) *Filesystem {
	return newFilesystem(client, bucket, opts...)
}

func newFilesystem(client ossClient, bucket string, opts ...Option) *Filesystem {
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
	result, err := s.client.GetObject(ctx, &aliyunoss.GetObjectRequest{
		Bucket: aliyunoss.Ptr(s.bucket),
		Key:    aliyunoss.Ptr(s.prefixer.Prefix(path)),
	})
	if err != nil {
		return nil, wrapPathError("open", path, err)
	}
	return result.Body, nil
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
	request := &aliyunoss.PutObjectRequest{
		Bucket:   aliyunoss.Ptr(s.bucket),
		Key:      aliyunoss.Ptr(s.prefixer.Prefix(path)),
		Body:     src,
		Metadata: cloneMetadata(options.Metadata),
	}
	if options.ContentType != "" {
		request.ContentType = aliyunoss.Ptr(options.ContentType)
	}
	_, err := s.client.PutObject(ctx, request)
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
	_, err := s.client.DeleteObject(ctx, &aliyunoss.DeleteObjectRequest{
		Bucket: aliyunoss.Ptr(s.bucket),
		Key:    aliyunoss.Ptr(s.prefixer.Prefix(path)),
	})
	if err != nil {
		return wrapPathError("delete", path, err)
	}
	return nil
}

// Stat returns object metadata. If an exact object does not exist, Stat issues
// one additional ListObjectsV2 request to detect a virtual directory prefix.
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
	result, err := s.client.HeadObject(ctx, &aliyunoss.HeadObjectRequest{
		Bucket: aliyunoss.Ptr(s.bucket),
		Key:    aliyunoss.Ptr(s.prefixer.Prefix(path)),
	})
	if err != nil {
		if isNotFound(err) {
			directory, listErr := s.hasChildren(ctx, path)
			if listErr != nil {
				return filesystem.Entry{}, wrapPathError("stat", path, listErr)
			}
			if directory {
				return filesystem.Entry{Path: path, Kind: filesystem.EntryKindDirectory}, nil
			}
		}
		return filesystem.Entry{}, wrapPathError("stat", path, err)
	}

	entry := filesystem.Entry{
		Path:        path,
		Kind:        filesystem.EntryKindFile,
		Size:        result.ContentLength,
		ContentType: dereference(result.ContentType),
		Metadata:    cloneMetadata(result.Metadata),
	}
	if result.LastModified != nil {
		entry.LastModified = *result.LastModified
	}
	return entry, nil
}

func (s *Filesystem) hasChildren(ctx context.Context, path string) (bool, error) {
	result, err := s.client.ListObjectsV2(ctx, &aliyunoss.ListObjectsV2Request{
		Bucket:  aliyunoss.Ptr(s.bucket),
		Prefix:  aliyunoss.Ptr(directoryPrefix(s.prefixer.Prefix(path))),
		MaxKeys: 1,
	})
	if err != nil {
		return false, err
	}
	return len(result.Contents) > 0 || len(result.CommonPrefixes) > 0, nil
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
	request := &aliyunoss.ListObjectsV2Request{
		Bucket: aliyunoss.Ptr(s.bucket),
		Prefix: aliyunoss.Ptr(directoryPrefix(s.prefixer.Prefix(path))),
	}
	if !options.Recursive {
		request.Delimiter = aliyunoss.Ptr("/")
	}
	if options.Cursor != "" {
		request.ContinuationToken = aliyunoss.Ptr(options.Cursor)
	}
	if options.Limit > 0 {
		request.MaxKeys = int32(min(options.Limit, 1000))
	}

	result, err := s.client.ListObjectsV2(ctx, request)
	if err != nil {
		return filesystem.ListPage{}, wrapPathError("list", path, err)
	}
	page := filesystem.ListPage{Entries: s.entries(result, options.Kind)}
	if result.NextContinuationToken != nil {
		page.NextCursor = *result.NextContinuationToken
	}
	return page, nil
}

// Copy copies src to dst using OSS's native server-side copy operation.
func (s *Filesystem) Copy(ctx context.Context, src, dst string) error {
	if err := filesystem.ValidateFilePath(src); err != nil {
		return err
	}
	if err := filesystem.ValidateFilePath(dst); err != nil {
		return err
	}
	_, err := s.client.CopyObject(ctx, &aliyunoss.CopyObjectRequest{
		Bucket:       aliyunoss.Ptr(s.bucket),
		Key:          aliyunoss.Ptr(s.prefixer.Prefix(dst)),
		SourceBucket: aliyunoss.Ptr(s.bucket),
		SourceKey:    aliyunoss.Ptr(s.prefixer.Prefix(src)),
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
	result *aliyunoss.ListObjectsV2Result,
	kind filesystem.EntryKind,
) []filesystem.Entry {
	byPath := make(map[string]filesystem.Entry, len(result.Contents)+len(result.CommonPrefixes))
	for _, object := range result.Contents {
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
		entry := filesystem.Entry{Path: path, Kind: entryKind, Size: object.Size}
		if object.LastModified != nil {
			entry.LastModified = *object.LastModified
		}
		byPath[path] = entry
	}
	for _, commonPrefix := range result.CommonPrefixes {
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
	var serviceError *aliyunoss.ServiceError
	if !errors.As(err, &serviceError) {
		return false
	}
	if serviceError.StatusCode == 404 {
		return true
	}
	switch serviceError.Code {
	case "NoSuchKey", "NoSuchObject", "NotFound":
		return true
	default:
		return false
	}
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
