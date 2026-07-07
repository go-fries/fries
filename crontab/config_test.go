package crontab

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLoggerConfig(t *testing.T) {
	t.Parallel()

	custom := slog.New(&recordingLogger{})

	tests := []struct {
		name string
		opts []LoggerOption
		want *slog.Logger
	}{
		{
			name: "defaults to slog default",
			want: slog.Default(),
		},
		{
			name: "uses custom logger",
			opts: []LoggerOption{WithLogger(custom)},
			want: custom,
		},
		{
			name: "ignores nil logger",
			opts: []LoggerOption{WithLogger(nil)},
			want: slog.Default(),
		},
		{
			name: "ignores nil option",
			opts: []LoggerOption{nil},
			want: slog.Default(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := newLoggerConfig(tt.opts...)

			assert.Same(t, tt.want, cfg.logger)
		})
	}
}

func TestNewServerConfig(t *testing.T) {
	t.Parallel()

	custom := slog.New(&recordingLogger{})

	tests := []struct {
		name string
		opts []ServerOption
		want *slog.Logger
	}{
		{
			name: "defaults to slog default",
			want: slog.Default(),
		},
		{
			name: "uses custom logger",
			opts: []ServerOption{WithLogger(custom)},
			want: custom,
		},
		{
			name: "ignores nil logger",
			opts: []ServerOption{WithLogger(nil)},
			want: slog.Default(),
		},
		{
			name: "ignores nil option",
			opts: []ServerOption{nil},
			want: slog.Default(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := newServerConfig(tt.opts...)

			assert.Same(t, tt.want, cfg.logger)
		})
	}
}
