package event

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDispatcherContext(t *testing.T) {
	dispatcher, ok := FromContext(t.Context())
	assert.False(t, ok)
	assert.Nil(t, dispatcher)

	expected := New()
	ctx := NewContext(t.Context(), expected)
	dispatcher, ok = FromContext(ctx)
	require.True(t, ok)
	assert.Same(t, expected, dispatcher)
}

func TestDispatcherContextValidation(t *testing.T) {
	assert.Panics(t, func() { NewContext(nil, New()) }) //nolint:staticcheck // Verifies the nil context contract.
	assert.Panics(t, func() { NewContext(t.Context(), nil) })
	assert.Panics(t, func() { FromContext(nil) }) //nolint:staticcheck // Verifies the nil context contract.
}
