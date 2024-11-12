package cache

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepository(t *testing.T) {
	repo := NewRepository(&NullStore{})
	ctx := context.Background()

	// -----store----

	// Has
	has, err := repo.Has(ctx, "test")
	assert.Nil(t, err)
	assert.True(t, has)

	// Get
	assert.Nil(t, repo.Get(ctx, "test", nil))

	// Put
	put, err := repo.Put(ctx, "test", "test", 0)
	assert.Nil(t, err)
	assert.True(t, put)

	// Increment
	incr, err := repo.Increment(ctx, "test", 1)
	assert.Nil(t, err)
	assert.Equal(t, 0, incr) // because of NullStore

	// Decrement
	decr, err := repo.Decrement(ctx, "test", 1)
	assert.Nil(t, err)
	assert.Equal(t, 0, decr) // because of NullStore

	// Forever
	forever, err := repo.Forever(ctx, "test", "test")
	assert.Nil(t, err)
	assert.True(t, forever)

	// Forget
	forget, err := repo.Forget(ctx, "test")
	assert.Nil(t, err)
	assert.True(t, forget)

	// Flush
	flush, err := repo.Flush(ctx)
	assert.Nil(t, err)
	assert.True(t, flush)

	// GetPrefix
	assert.Empty(t, repo.GetPrefix())

	// Lock
	locker := repo.Lock("test", 0)
	assert.Nil(t, locker) // because of NullStore

	// -----repository----

	// missing
	missing, err := repo.Missing(ctx, "test")
	assert.Nil(t, err)
	assert.False(t, missing)

	// Add
	add, err := repo.Add(ctx, "test", "test", 0)
	assert.Nil(t, err)
	assert.False(t, add) // because of NullStore

	// Delete
	del, err := repo.Delete(ctx, "test")
	assert.Nil(t, err)
	assert.True(t, del)

	// Set
	set, err := repo.Set(ctx, "test", "test", 0)
	assert.Nil(t, err)
	assert.True(t, set)
}
