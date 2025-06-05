package cached

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"hash"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerator_UniquenessAndDifference(t *testing.T) {
	ctx := context.Background()
	opt := GeneratorOptions{}

	for _, tt := range []struct {
		name      string
		generator Generator
	}{
		{"SimpleGenerator", NewSimpleGenerator()},
		{"HashGenerator", NewHashGenerator(sha256.New())},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("Generate uniqueness", func(t *testing.T) {
				for _, tt := range []struct {
					callback func() string
				}{
					{func() string { return tt.generator.Generate(ctx, "foo", opt) }},
					{func() string { return tt.generator.Generate(ctx, "foo", GeneratorOptions{Model: "bar"}) }},
				} {
					assert.Equal(t, tt.callback(), tt.callback())
				}
			})

			t.Run("Generate different keys", func(t *testing.T) {
				assert.NotEqual(t, tt.generator.Generate(ctx, "foo", opt), tt.generator.Generate(ctx, "bar", opt))
				assert.NotEqual(t, tt.generator.Generate(ctx, "foo", opt), tt.generator.Generate(ctx, "foo", GeneratorOptions{Model: "bar"}))
				assert.NotEqual(t, tt.generator.Generate(ctx, "foo", GeneratorOptions{Model: "bar"}),
					tt.generator.Generate(ctx, "foo", GeneratorOptions{Model: "baz"}))
			})
		})
	}
}

func TestGenerator_SimpleGenerator(t *testing.T) {
	text := "test text"
	model := "test-model"
	ctx := context.Background()
	opt := GeneratorOptions{}

	generator := NewSimpleGenerator()
	assert.Equal(t, generator.Generate(ctx, text, GeneratorOptions{Model: model}), text+"-"+model)
	assert.Equal(t, generator.Generate(ctx, text, opt), text+"-")
	assert.Equal(t, generator.Generate(ctx, "", opt), "-")
}

func TestGenerator_HashGenerator(t *testing.T) {
	text := "test text"
	model := "test-model"
	ctx := context.Background()

	for _, tt := range []struct {
		name string
		hash hash.Hash
	}{
		{"sha256", sha256.New()},
		{"md5", md5.New()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewHashGenerator(tt.hash)
			assert.NotEmpty(t, generator.Generate(ctx, text, GeneratorOptions{Model: model}))
		})
	}
}
