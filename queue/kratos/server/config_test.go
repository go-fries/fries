package server

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	t.Parallel()

	custom := slog.New(&recordingHandler{})

	tests := []struct {
		name       string
		opts       []Option
		wantLogger *slog.Logger
	}{
		{
			name:       "defaults",
			wantLogger: slog.Default(),
		},
		{
			name:       "sets logger",
			opts:       []Option{WithLogger(custom)},
			wantLogger: custom,
		},
		{
			name:       "ignores nil logger",
			opts:       []Option{WithLogger(nil)},
			wantLogger: slog.Default(),
		},
		{
			name:       "ignores nil option",
			opts:       []Option{nil},
			wantLogger: slog.Default(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := newConfig(tt.opts...)

			require.NotNil(t, cfg)
			assert.Same(t, tt.wantLogger, cfg.logger)
		})
	}
}
