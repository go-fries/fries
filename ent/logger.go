package ent

import (
	"context"
	"fmt"
	"log/slog"
)

type Logger func(...any)

func NewLogger(logger *slog.Logger, levels ...slog.Level) Logger {
	if logger == nil {
		logger = slog.Default()
	}

	level := slog.LevelDebug
	if len(levels) > 0 {
		level = levels[0]
	}

	return func(args ...any) {
		logger.Log(context.Background(), level, fmt.Sprint(args...))
	}
}
