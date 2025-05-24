// provider_test.go
package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testConfig struct {
	Name string
	Port int
}

func TestProvider(t *testing.T) {
	cfg := testConfig{Name: "test", Port: 1234}
	provider := NewProvider(cfg)

	ctx := context.Background()
	newCtx, err := provider.Bootstrap(ctx)
	require.NoError(t, err)

	got, ok := FromContext[testConfig](newCtx)
	assert.True(t, ok)
	assert.Equal(t, cfg, got)

	terminatedCtx, err := provider.Terminate(newCtx)
	require.NoError(t, err)
	assert.Equal(t, newCtx, terminatedCtx)
}
