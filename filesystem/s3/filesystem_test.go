package s3

import (
	"context"
	"testing"
	"time"

	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-fries/fries/filesystem/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilesystemList(t *testing.T) {
	modified := time.Now()
	client := &fakeClient{
		listOutput: &awss3.ListObjectsV2Output{
			Contents: []types.Object{{
				Key:          ptr("root/dir/file.txt"),
				Size:         ptr(int64(7)),
				LastModified: &modified,
			}},
			CommonPrefixes:        []types.CommonPrefix{{Prefix: ptr("root/dir/nested/")}},
			NextContinuationToken: ptr("next"),
		},
	}
	storage := newFilesystem(client, "bucket", WithRoot("root"))

	page, err := storage.List(t.Context(), "dir", filesystem.ListOptions{
		Limit:  2,
		Cursor: "cursor",
	})
	require.NoError(t, err)
	assert.Equal(t, "root/dir/", dereference(client.listInput.Prefix))
	assert.Equal(t, "/", dereference(client.listInput.Delimiter))
	assert.Equal(t, "cursor", dereference(client.listInput.ContinuationToken))
	assert.Equal(t, []string{"dir/file.txt", "dir/nested"}, entryPaths(page.Entries))
	assert.Equal(t, "next", page.NextCursor)
}

func TestFilesystemMove(t *testing.T) {
	client := &fakeClient{}
	storage := newFilesystem(client, "bucket", WithRoot("root"))

	require.NoError(t, storage.Move(t.Context(), "source file.txt", "target.txt"))
	require.NotNil(t, client.copyInput)
	require.NotNil(t, client.deleteInput)
	assert.Equal(t, "root/target.txt", dereference(client.copyInput.Key))
	assert.Equal(t, "root/source file.txt", dereference(client.deleteInput.Key))
}

func TestFilesystemNotFound(t *testing.T) {
	client := &fakeClient{getErr: &types.NoSuchKey{}}
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
		headErr: &types.NoSuchKey{},
		listOutput: &awss3.ListObjectsV2Output{
			Contents: []types.Object{{Key: ptr("root/images/file.jpg")}},
		},
	}
	storage := newFilesystem(client, "bucket", WithRoot("root"))

	entry, err := storage.Stat(t.Context(), "images")
	require.NoError(t, err)
	assert.Equal(t, filesystem.Entry{Path: "images", Kind: filesystem.EntryKindDirectory}, entry)
	assert.Equal(t, "root/images/", dereference(client.listInput.Prefix))
	assert.Equal(t, int32(1), *client.listInput.MaxKeys)
}

func TestFilesystemStatMissing(t *testing.T) {
	client := &fakeClient{
		headErr:    &types.NoSuchKey{},
		listOutput: &awss3.ListObjectsV2Output{},
	}
	storage := newFilesystem(client, "bucket")

	_, err := storage.Stat(t.Context(), "missing")
	assert.ErrorIs(t, err, filesystem.ErrNotFound)
}

type fakeClient struct {
	getErr      error
	headErr     error
	headOutput  *awss3.HeadObjectOutput
	listInput   *awss3.ListObjectsV2Input
	listOutput  *awss3.ListObjectsV2Output
	copyInput   *awss3.CopyObjectInput
	deleteInput *awss3.DeleteObjectInput
}

func (c *fakeClient) GetObject(
	context.Context,
	*awss3.GetObjectInput,
	...func(*awss3.Options),
) (*awss3.GetObjectOutput, error) {
	return nil, c.getErr
}

func (*fakeClient) PutObject(
	context.Context,
	*awss3.PutObjectInput,
	...func(*awss3.Options),
) (*awss3.PutObjectOutput, error) {
	return &awss3.PutObjectOutput{}, nil
}

func (c *fakeClient) DeleteObject(
	_ context.Context,
	input *awss3.DeleteObjectInput,
	_ ...func(*awss3.Options),
) (*awss3.DeleteObjectOutput, error) {
	c.deleteInput = input
	return &awss3.DeleteObjectOutput{}, nil
}

func (c *fakeClient) HeadObject(
	_ context.Context,
	_ *awss3.HeadObjectInput,
	_ ...func(*awss3.Options),
) (*awss3.HeadObjectOutput, error) {
	if c.headOutput == nil {
		c.headOutput = &awss3.HeadObjectOutput{}
	}
	return c.headOutput, c.headErr
}

func (c *fakeClient) ListObjectsV2(
	_ context.Context,
	input *awss3.ListObjectsV2Input,
	_ ...func(*awss3.Options),
) (*awss3.ListObjectsV2Output, error) {
	c.listInput = input
	return c.listOutput, nil
}

func (c *fakeClient) CopyObject(
	_ context.Context,
	input *awss3.CopyObjectInput,
	_ ...func(*awss3.Options),
) (*awss3.CopyObjectOutput, error) {
	c.copyInput = input
	return &awss3.CopyObjectOutput{}, nil
}

func entryPaths(entries []filesystem.Entry) []string {
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		paths = append(paths, entry.Path)
	}
	return paths
}
