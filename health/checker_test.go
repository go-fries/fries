package health_test

import (
	"context"
	"testing"

	"github.com/go-fries/fries/health/v4"
	"github.com/stretchr/testify/assert"
)

func TestCheckFunc(t *testing.T) {
	called := false
	checker := health.CheckFunc(func(ctx context.Context) error {
		called = true
		assert.Same(t, t.Context(), ctx)
		return assert.AnError
	})

	err := checker.Check(t.Context())

	assert.ErrorIs(t, err, assert.AnError)
	assert.True(t, called)
}
