package contexts

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type (
	mockProviderStruct1 struct{}
	mockProviderStruct2 struct{}
	mockProviderStruct3 struct{}
)

var result chan string

var (
	mockProvider1 = func(ctx context.Context) (context.Context, error) {
		result <- "mockProvider1"
		return context.WithValue(ctx, mockProviderStruct1{}, "mockProvider1"), nil
	}

	mockProvider2 = func(ctx context.Context) (context.Context, error) {
		result <- "mockProvider2"
		return context.WithValue(ctx, mockProviderStruct2{}, "mockProvider2"), nil
	}

	mockProvider3 = func(ctx context.Context) (context.Context, error) {
		result <- "mockProvider3"
		return ctx, errors.New("mockProvider3")
	}
)

func TestPipe(t *testing.T) {
	result = make(chan string, 2)
	ctx1, err1 := Pipe(
		context.Background(),
		mockProvider1, mockProvider2,
	)
	assert.NoError(t, err1)
	assert.Equal(t, "mockProvider1", ctx1.Value(mockProviderStruct1{}))
	assert.Equal(t, "mockProvider2", ctx1.Value(mockProviderStruct2{}))
	assert.Equal(t, "mockProvider1", <-result)
	assert.Equal(t, "mockProvider2", <-result)

	ctx2, err2 := Pipe(
		context.Background(),
		mockProvider1, mockProvider3,
	)
	assert.Error(t, err2)
	assert.NotNil(t, ctx2)
	assert.Equal(t, "mockProvider1", ctx2.Value(mockProviderStruct1{}))
	assert.Nil(t, ctx2.Value(mockProviderStruct3{}))
	assert.Equal(t, "mockProvider1", <-result)
}

func TestChain(t *testing.T) {
	result = make(chan string, 2)
	ctx1, err1 := Chain(
		context.Background(),
		mockProvider1, mockProvider2,
	)
	assert.NoError(t, err1)
	assert.Equal(t, "mockProvider1", ctx1.Value(mockProviderStruct1{}))
	assert.Equal(t, "mockProvider2", ctx1.Value(mockProviderStruct2{}))
	assert.Equal(t, "mockProvider2", <-result)
	assert.Equal(t, "mockProvider1", <-result)

	ctx2, err2 := Chain(
		context.Background(),
		mockProvider3, mockProvider1,
	)
	assert.Error(t, err2)
	assert.Equal(t, "mockProvider1", ctx2.Value(mockProviderStruct1{}))
	assert.Equal(t, "mockProvider1", <-result)
}
