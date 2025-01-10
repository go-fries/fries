package contexts

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValue(t *testing.T) {
	ctx := context.Background()

	type (
		key1 struct{}
		key2 struct{}
		key3 struct{}
		key4 struct{}
	)

	ctx1 := context.WithValue(ctx, key1{}, "test")
	assert.Equal(t, "test", MustValue[string](ctx1, key1{}))

	ctx2 := context.WithValue(ctx, key2{}, 1)
	assert.Equal(t, 1, MustValue[int](ctx2, key2{}))

	ctx3 := context.WithValue(ctx, key3{}, 1.1)
	assert.Equal(t, 1.1, MustValue[float64](ctx3, key3{}))

	assert.Panics(t, func() {
		MustValue[string](ctx, key4{})
	})
}
