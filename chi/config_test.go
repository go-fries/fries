package chi

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type customOption struct {
	logger *slog.Logger
}

func (o customOption) apply(cfg *config) {
	cfg.logger = o.logger
}

func TestNewConfig(t *testing.T) {
	t.Parallel()

	custom := slog.New(&recordingHandler{})

	tests := []struct {
		name       string
		opts       []Option
		wantAddr   string
		wantLogger *slog.Logger
	}{
		{
			name:       "defaults",
			wantAddr:   ":8080",
			wantLogger: slog.Default(),
		},
		{
			name:       "sets addr",
			opts:       []Option{WithAddr(":9000")},
			wantAddr:   ":9000",
			wantLogger: slog.Default(),
		},
		{
			name:       "sets logger",
			opts:       []Option{WithLogger(custom)},
			wantAddr:   ":8080",
			wantLogger: custom,
		},
		{
			name:       "supports custom option",
			opts:       []Option{customOption{logger: custom}},
			wantAddr:   ":8080",
			wantLogger: custom,
		},
		{
			name:       "ignores nil logger",
			opts:       []Option{WithLogger(nil)},
			wantAddr:   ":8080",
			wantLogger: slog.Default(),
		},
		{
			name:       "ignores nil option",
			opts:       []Option{nil},
			wantAddr:   ":8080",
			wantLogger: slog.Default(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := newConfig(tt.opts...)

			require.NotNil(t, cfg)
			assert.Equal(t, tt.wantAddr, cfg.addr)
			assert.Same(t, tt.wantLogger, cfg.logger)
		})
	}
}
