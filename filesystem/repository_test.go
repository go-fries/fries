package filesystem

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoopRepository(t *testing.T) {
	repo := NewRepository(NoopFilesystem{})
	ctx := context.Background()

	assert.NoError(t, repo.Put(ctx, "noop", []byte("noop")))
	assert.NoError(t, repo.Destroy(ctx, "noop"))

	exists, err := repo.Exists(ctx, "noop")
	assert.NoError(t, err)
	assert.True(t, exists)

	missing, err := repo.Missing(ctx, "noop")
	assert.NoError(t, err)
	assert.False(t, missing)

	assert.NoError(t, repo.Rename(ctx, "noop", "noop2"))

	assert.NoError(t, repo.Prepend(ctx, "noop", []byte("noop")))
	assert.NoError(t, repo.Append(ctx, "noop", []byte("noop")))
	assert.NoError(t, repo.Copy(ctx, "noop", "noop2"))
}
