package oss

import (
	"context"
	"testing"
	"time"

	aliyunoss "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/go-fries/fries/filesystem/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilesystemList(t *testing.T) {
	modified := time.Now()
	client := &fakeClient{
		listResult: &aliyunoss.ListObjectsV2Result{
			Contents: []aliyunoss.ObjectProperties{
				{
					Key:          aliyunoss.Ptr("root/dir/file.txt"),
					Size:         7,
					LastModified: &modified,
				},
				{Key: aliyunoss.Ptr("root/dir/marker/")},
			},
			CommonPrefixes:        []aliyunoss.CommonPrefix{{Prefix: aliyunoss.Ptr("root/dir/nested/")}},
			NextContinuationToken: aliyunoss.Ptr("next"),
		},
	}
	storage := newFilesystem(client, "bucket", WithRoot("root"))

	page, err := storage.List(t.Context(), "dir", filesystem.ListOptions{
		Limit:  2,
		Cursor: "cursor",
	})
	require.NoError(t, err)
	assert.Equal(t, "root/dir/", dereference(client.listRequest.Prefix))
	assert.Equal(t, "/", dereference(client.listRequest.Delimiter))
	assert.Equal(t, "cursor", dereference(client.listRequest.ContinuationToken))
	assert.Equal(t, []string{"dir/file.txt"}, entryPaths(page.Entries))
	assert.Equal(t, "next", page.NextCursor)

	directories, err := storage.List(t.Context(), "dir", filesystem.ListOptions{
		Kind: filesystem.EntryKindDirectory,
	})
	require.NoError(t, err)
	assert.Empty(t, directories.Entries)
}

func TestFilesystemMove(t *testing.T) {
	client := &fakeClient{}
	storage := newFilesystem(client, "bucket", WithRoot("root"))

	require.NoError(t, storage.Move(t.Context(), "source.txt", "target.txt"))
	require.NotNil(t, client.copyRequest)
	require.NotNil(t, client.deleteRequest)
	assert.Equal(t, "root/source.txt", dereference(client.copyRequest.SourceKey))
	assert.Equal(t, "root/target.txt", dereference(client.copyRequest.Key))
	assert.Equal(t, "root/source.txt", dereference(client.deleteRequest.Key))
}

func TestFilesystemNotFound(t *testing.T) {
	client := &fakeClient{getErr: &aliyunoss.ServiceError{StatusCode: 404, Code: "NoSuchKey"}}
	storage := newFilesystem(client, "bucket")

	_, err := storage.Open(t.Context(), "missing.txt")
	assert.ErrorIs(t, err, filesystem.ErrNotFound)
}

func TestFilesystemRoot(t *testing.T) {
	storage := newFilesystem(&fakeClient{}, "bucket", WithRoot("root"))

	entry, err := storage.Stat(t.Context(), ".")
	require.NoError(t, err)
	assert.Equal(t, filesystem.Entry{Path: ".", Kind: filesystem.EntryKindDirectory}, entry)

	_, err = storage.Open(t.Context(), ".")
	assert.ErrorIs(t, err, filesystem.ErrInvalidPath)
	assert.ErrorIs(t, storage.Delete(t.Context(), "."), filesystem.ErrInvalidPath)
	assert.ErrorIs(t, storage.Copy(t.Context(), ".", "target.txt"), filesystem.ErrInvalidPath)
}

func TestFilesystemStatVirtualDirectory(t *testing.T) {
	client := &fakeClient{
		headErr: &aliyunoss.ServiceError{StatusCode: 404, Code: "NoSuchKey"},
		listResult: &aliyunoss.ListObjectsV2Result{
			Contents: []aliyunoss.ObjectProperties{{Key: aliyunoss.Ptr("root/images/file.jpg")}},
		},
	}
	storage := newFilesystem(client, "bucket", WithRoot("root"))

	entry, err := storage.Stat(t.Context(), "images")
	require.NoError(t, err)
	assert.Equal(t, filesystem.Entry{Path: "images", Kind: filesystem.EntryKindDirectory}, entry)
	assert.Equal(t, "root/images/", dereference(client.listRequest.Prefix))
	assert.Equal(t, int32(1), client.listRequest.MaxKeys)
}

func TestFilesystemStatMissing(t *testing.T) {
	client := &fakeClient{
		headErr:    &aliyunoss.ServiceError{StatusCode: 404, Code: "NoSuchKey"},
		listResult: &aliyunoss.ListObjectsV2Result{},
	}
	storage := newFilesystem(client, "bucket")

	_, err := storage.Stat(t.Context(), "missing")
	assert.ErrorIs(t, err, filesystem.ErrNotFound)
}

type fakeClient struct {
	getErr        error
	headErr       error
	headResult    *aliyunoss.HeadObjectResult
	listRequest   *aliyunoss.ListObjectsV2Request
	listResult    *aliyunoss.ListObjectsV2Result
	copyRequest   *aliyunoss.CopyObjectRequest
	deleteRequest *aliyunoss.DeleteObjectRequest
}

func (c *fakeClient) GetObject(
	context.Context,
	*aliyunoss.GetObjectRequest,
	...func(*aliyunoss.Options),
) (*aliyunoss.GetObjectResult, error) {
	return nil, c.getErr
}

func (*fakeClient) PutObject(
	context.Context,
	*aliyunoss.PutObjectRequest,
	...func(*aliyunoss.Options),
) (*aliyunoss.PutObjectResult, error) {
	return &aliyunoss.PutObjectResult{}, nil
}

func (c *fakeClient) DeleteObject(
	_ context.Context,
	request *aliyunoss.DeleteObjectRequest,
	_ ...func(*aliyunoss.Options),
) (*aliyunoss.DeleteObjectResult, error) {
	c.deleteRequest = request
	return &aliyunoss.DeleteObjectResult{}, nil
}

func (c *fakeClient) HeadObject(
	_ context.Context,
	_ *aliyunoss.HeadObjectRequest,
	_ ...func(*aliyunoss.Options),
) (*aliyunoss.HeadObjectResult, error) {
	if c.headResult == nil {
		c.headResult = &aliyunoss.HeadObjectResult{}
	}
	return c.headResult, c.headErr
}

func (c *fakeClient) ListObjectsV2(
	_ context.Context,
	request *aliyunoss.ListObjectsV2Request,
	_ ...func(*aliyunoss.Options),
) (*aliyunoss.ListObjectsV2Result, error) {
	c.listRequest = request
	return c.listResult, nil
}

func (c *fakeClient) CopyObject(
	_ context.Context,
	request *aliyunoss.CopyObjectRequest,
	_ ...func(*aliyunoss.Options),
) (*aliyunoss.CopyObjectResult, error) {
	c.copyRequest = request
	return &aliyunoss.CopyObjectResult{}, nil
}

func entryPaths(entries []filesystem.Entry) []string {
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		paths = append(paths, entry.Path)
	}
	return paths
}
