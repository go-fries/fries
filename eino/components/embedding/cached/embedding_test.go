package cached

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockEmbedder struct {
	embedding.Embedder
	mock.Mock
}

func (m *mockEmbedder) EmbedStrings(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
	args := m.Called(ctx, texts, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([][]float64), args.Error(1)
}

type mockCacher struct {
	Cacher
	mock.Mock
}

var _ Cacher = (*mockCacher)(nil)

func (m *mockCacher) Get(ctx context.Context, key string) ([]float64, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]float64), args.Error(1)
}

func (m *mockCacher) Set(ctx context.Context, key string, value []float64, expire time.Duration) error {
	args := m.Called(ctx, key, value, expire)
	return args.Error(0)
}

func TestEmbedder_EmbedStrings(t *testing.T) {
	ctx := t.Context()
	texts := []string{"foo", "bar"}
	embeddings := [][]float64{{1.1, 2.2}, {3.3, 4.4}}
	expiration := time.Minute
	generatorOpt := GeneratorOptions{}

	t.Run("embedder not set cacher", func(t *testing.T) {
		me := new(mockEmbedder)
		e := NewEmbedder(me, WithGenerator(NewSimpleGenerator()))

		assert.Equal(t, defaultCacher, e.cacher)
		me.AssertExpectations(t)
	})

	t.Run("embedder not set generator", func(t *testing.T) {
		mc := new(mockCacher)
		me := new(mockEmbedder)
		e := NewEmbedder(me, WithCacher(mc))

		assert.Equal(t, defaultGenerator, e.generator)
		mc.AssertExpectations(t)
		me.AssertExpectations(t)
	})

	t.Run("all cache hit", func(t *testing.T) {
		mc := new(mockCacher)
		me := new(mockEmbedder)
		e := NewEmbedder(me, WithCacher(mc), WithGenerator(NewSimpleGenerator()), WithExpiration(expiration))

		for i, text := range texts {
			key := e.generator.Generate(ctx, text, generatorOpt)
			mc.On("Get", mock.Anything, key).Return(embeddings[i], nil)
		}

		result, err := e.EmbedStrings(ctx, texts)
		assert.NoError(t, err)
		assert.Equal(t, embeddings, result)
		mc.AssertExpectations(t)
	})

	t.Run("partial cache hit", func(t *testing.T) {
		mc := new(mockCacher)
		me := new(mockEmbedder)
		e := NewEmbedder(me, WithCacher(mc), WithGenerator(NewSimpleGenerator()), WithExpiration(expiration))

		key0 := e.generator.Generate(ctx, texts[0], generatorOpt)
		key1 := e.generator.Generate(ctx, texts[1], generatorOpt)

		mc.On("Get", mock.Anything, key0).Return(nil, ErrCacherKeyNotFound)
		mc.On("Get", mock.Anything, key1).Return(embeddings[1], nil)
		me.On("EmbedStrings", mock.Anything, []string{texts[0]}, mock.Anything).Return([][]float64{embeddings[0]}, nil)
		mc.On("Set", mock.Anything, key0, embeddings[0], expiration).Return(nil)

		result, err := e.EmbedStrings(ctx, texts)
		assert.NoError(t, err)
		assert.Equal(t, embeddings, result)

		mc.AssertExpectations(t)
		me.AssertExpectations(t)
	})

	t.Run("all cache miss", func(t *testing.T) {
		mc := new(mockCacher)
		me := new(mockEmbedder)
		e := NewEmbedder(me, WithCacher(mc), WithGenerator(NewSimpleGenerator()), WithExpiration(expiration))

		key0 := e.generator.Generate(ctx, texts[0], generatorOpt)
		key1 := e.generator.Generate(ctx, texts[1], generatorOpt)

		mc.On("Get", mock.Anything, key0).Return(nil, ErrCacherKeyNotFound)
		mc.On("Get", mock.Anything, key1).Return(nil, ErrCacherKeyNotFound)
		me.On("EmbedStrings", mock.Anything, texts, mock.Anything).Return(embeddings, nil)
		mc.On("Set", mock.Anything, key0, embeddings[0], expiration).Return(nil)
		mc.On("Set", mock.Anything, key1, embeddings[1], expiration).Return(nil)

		result, err := e.EmbedStrings(ctx, texts)
		assert.NoError(t, err)
		assert.Equal(t, embeddings, result)
		mc.AssertExpectations(t)
		me.AssertExpectations(t)
	})

	t.Run("cache get error", func(t *testing.T) {
		mc := new(mockCacher)
		me := new(mockEmbedder)
		e := NewEmbedder(me, WithCacher(mc))

		key := e.generator.Generate(ctx, texts[0], generatorOpt)
		mc.On("Get", mock.Anything, key).Return(nil, errors.New("cache error"))

		_, err := e.EmbedStrings(ctx, []string{texts[0]})
		assert.Error(t, err)
		mc.AssertExpectations(t)
		me.AssertExpectations(t)
	})

	t.Run("underlying embedder error", func(t *testing.T) {
		mc := new(mockCacher)
		me := new(mockEmbedder)
		e := NewEmbedder(me, WithCacher(mc), WithGenerator(NewSimpleGenerator()), WithExpiration(expiration))

		key := e.generator.Generate(ctx, texts[0], generatorOpt)
		mc.On("Get", mock.Anything, key).Return(nil, ErrCacherKeyNotFound)
		me.On("EmbedStrings", mock.Anything, []string{texts[0]}, mock.Anything).Return(nil, errors.New("embed error"))

		_, err := e.EmbedStrings(ctx, []string{texts[0]})
		assert.Error(t, err)
		mc.AssertExpectations(t)
		me.AssertExpectations(t)
	})

	t.Run("cache set error, ignore", func(t *testing.T) {
		mc := new(mockCacher)
		me := new(mockEmbedder)
		e := NewEmbedder(me, WithCacher(mc), WithGenerator(NewSimpleGenerator()), WithExpiration(expiration))

		key0 := e.generator.Generate(ctx, texts[0], generatorOpt)

		mc.On("Get", mock.Anything, key0).Return(nil, ErrCacherKeyNotFound)
		me.On("EmbedStrings", mock.Anything, []string{texts[0]}, mock.Anything).Return([][]float64{embeddings[0]}, nil)
		mc.On("Set", mock.Anything, key0, embeddings[0], expiration).Return(errors.New("set error"))

		result, err := e.EmbedStrings(ctx, []string{texts[0]})
		assert.NoError(t, err)
		assert.Equal(t, [][]float64{embeddings[0]}, result)
		mc.AssertExpectations(t)
		me.AssertExpectations(t)
	})
}

type contextMockCacher struct {
	t        *testing.T
	getTexts chan string
	setTexts chan string
}

func (c *contextMockCacher) Get(ctx context.Context, _ string) ([]float64, error) {
	text, ok := TextFromContext(ctx)
	assert.True(c.t, ok)
	c.getTexts <- text
	return nil, ErrCacherKeyNotFound
}

func (c *contextMockCacher) Set(ctx context.Context, _ string, _ []float64, _ time.Duration) error {
	text, ok := TextFromContext(ctx)
	assert.True(c.t, ok)
	c.setTexts <- text
	return nil
}

func TestEmbedder_ContextWithText(t *testing.T) {
	ctx := t.Context()
	embeddings := [][]float64{{1.1, 2.2}, {3.3, 4.4}}

	mc := &contextMockCacher{
		t:        t,
		getTexts: make(chan string, 2),
		setTexts: make(chan string, 2),
	}
	me := new(mockEmbedder)
	e := NewEmbedder(me, WithCacher(mc))

	texts := []string{"foo", "bar"}

	me.On("EmbedStrings", mock.Anything, texts, mock.Anything).Return(embeddings, nil)

	result, err := e.EmbedStrings(ctx, texts)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(mc.getTexts))
	assert.Equal(t, 2, len(mc.setTexts))
	assert.Equal(t, "foo", <-mc.getTexts)
	assert.Equal(t, "bar", <-mc.getTexts)
	assert.Equal(t, "foo", <-mc.setTexts)
	assert.Equal(t, "bar", <-mc.setTexts)
	assert.Equal(t, embeddings, result)

	me.AssertExpectations(t)
}
