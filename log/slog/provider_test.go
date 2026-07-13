package slog

import (
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProvider(t *testing.T) {
	previous := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(previous)
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	provider := NewProvider(logger)

	ctx := t.Context()
	bootstrapped, err := provider.Bootstrap(ctx)
	assert.NoError(t, err)
	assert.Equal(t, ctx, bootstrapped)
	assert.Same(t, logger, slog.Default())

	terminated, err := provider.Terminate(bootstrapped)
	assert.NoError(t, err)
	assert.Equal(t, bootstrapped, terminated)
	assert.Same(t, logger, slog.Default())
}
