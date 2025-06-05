package cached

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCacher_noCacher(t *testing.T) {
	c := &noCacher{}
	ctx := context.Background()

	assert.NoError(t, c.Set(ctx, "key", []float64{1.0, 2.0}, 0))
	value, err := c.Get(ctx, "key")
	assert.Error(t, err)
	assert.Equal(t, ErrCacherKeyNotFound, err)
	assert.Nil(t, value)
}
