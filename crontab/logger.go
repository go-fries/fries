package crontab

import (
	"context"
	"fmt"
	"log/slog"
)

// Logger adapts a slog logger to cron's Printf-compatible logger shape.
type Logger struct {
	logger *slog.Logger
}

// NewLogger creates a cron-compatible Logger.
//
// NewLogger uses slog's default logger unless opts replace it.
func NewLogger(opts ...LoggerOption) *Logger {
	c := &loggerConfig{
		logger: slog.Default(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt.applyLogger(c)
		}
	}
	return &Logger{logger: c.logger}
}

// Printf writes a formatted message at info level.
func (l *Logger) Printf(format string, v ...any) {
	l.logger.Log(context.Background(), slog.LevelInfo, fmt.Sprintf(format, v...))
}
