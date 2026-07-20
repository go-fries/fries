package response_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/go-fries/fries/http/response/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorMapperFunc(t *testing.T) {
	t.Parallel()

	type contextKey struct{}
	ctx := context.WithValue(t.Context(), contextKey{}, "value")
	targetErr := errors.New("not found")
	called := false

	mapper := response.ErrorMapperFunc(func(
		gotCtx context.Context,
		gotErr error,
	) (int, response.Body) {
		called = true
		assert.Same(t, ctx, gotCtx)
		assert.ErrorIs(t, gotErr, targetErr)
		return http.StatusNotFound, response.Failure(
			"Resource not found.",
			response.WithCode(10404),
		)
	})

	httpStatus, body := mapper.Map(ctx, targetErr)

	assert.True(t, called)
	assert.Equal(t, http.StatusNotFound, httpStatus)
	assert.False(t, body.Status)
	assert.Equal(t, "Resource not found.", body.Message)
	require.NotNil(t, body.Code)
	assert.Equal(t, 10404, *body.Code)
}

func TestErrorMapperFuncAcceptsNilError(t *testing.T) {
	t.Parallel()

	mapper := response.ErrorMapperFunc(func(
		_ context.Context,
		err error,
	) (int, response.Body) {
		return http.StatusOK, response.FromError(err)
	})

	httpStatus, body := mapper.Map(t.Context(), nil)

	assert.Equal(t, http.StatusOK, httpStatus)
	assert.True(t, body.Status)
	assert.Empty(t, body.Message)
}

func TestErrorMapperFuncPanicsWhenNil(t *testing.T) {
	t.Parallel()

	var mapper response.ErrorMapperFunc

	assert.PanicsWithValue(
		t,
		"response: nil error mapper function",
		func() {
			_, _ = mapper.Map(t.Context(), errors.New("failure"))
		},
	)
}

var _ response.ErrorMapper = response.ErrorMapperFunc(nil)
