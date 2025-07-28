package cached

import (
	"context"
	"crypto/sha256"
	"fmt"
	"hash"
	"sync"
)

var defaultGenerator Generator = NewHashGenerator(sha256.New)

// GeneratorOptions holds options for generating unique keys.
type GeneratorOptions struct {
	Model string
}

// Generator is an interface for generating unique keys based on text and optional embedding options.
// It is used to create cache keys for embedding results.
type Generator interface {
	Generate(ctx context.Context, text string, opt GeneratorOptions) string
}

// SimpleGenerator is a concrete implementation of the Generator interface that generates
// a simple key by concatenating the text and model without hashing.
type SimpleGenerator struct{}

var _ Generator = (*SimpleGenerator)(nil)

// NewSimpleGenerator creates a new [SimpleGenerator] instance.
func NewSimpleGenerator() *SimpleGenerator {
	return &SimpleGenerator{}
}

func (g *SimpleGenerator) Generate(_ context.Context, text string, opts GeneratorOptions) string {
	return fmt.Sprintf("%s-%s", text, opts.Model)
}

// HashGenerator is a concrete implementation of the [Generator] interface that uses a hasher function
// to generate a unique key based on the provided text and optional embedding options.
// It wraps a [SimpleGenerator] and applies a hasher function to the generated key.
//
// Note: Because of the use of the [hash.Hash] algorithm, there is a probability that data
// with different text and options will generate the same key. This is a trade-off
// between uniqueness and performance. If you need guaranteed uniqueness, consider
// using a different generator or a more complex hashing strategy.
type HashGenerator struct {
	*SimpleGenerator
	pool sync.Pool
}

type Hasher func() hash.Hash

var _ Generator = (*HashGenerator)(nil)

// NewHashGenerator creates a new [HashGenerator] with the specified hasher function.
func NewHashGenerator(hasher Hasher) *HashGenerator {
	return &HashGenerator{
		SimpleGenerator: NewSimpleGenerator(),
		pool: sync.Pool{
			New: func() any {
				return hasher()
			},
		},
	}
}

func (g *HashGenerator) Generate(ctx context.Context, text string, opts GeneratorOptions) string {
	plainText := g.SimpleGenerator.Generate(ctx, text, opts)
	hasher := g.pool.Get().(hash.Hash)
	defer func() {
		hasher.Reset()
		g.pool.Put(hasher)
	}()
	_, _ = hasher.Write([]byte(plainText))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}
