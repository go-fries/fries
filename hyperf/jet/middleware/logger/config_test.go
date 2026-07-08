package logger

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
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
		name string
		opts []Option
		want *slog.Logger
	}{
		{
			name: "defaults to slog default",
			want: slog.Default(),
		},
		{
			name: "uses custom logger",
			opts: []Option{WithLogger(custom)},
			want: custom,
		},
		{
			name: "uses custom option",
			opts: []Option{customOption{logger: custom}},
			want: custom,
		},
		{
			name: "ignores nil logger",
			opts: []Option{WithLogger(nil)},
			want: slog.Default(),
		},
		{
			name: "ignores nil option",
			opts: []Option{nil},
			want: slog.Default(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := newConfig(tt.opts...)

			assert.Same(t, tt.want, cfg.logger)
		})
	}
}
