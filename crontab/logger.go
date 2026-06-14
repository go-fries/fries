package crontab

import (
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
)

type loggerConfig struct {
	logger log.Logger
}

// LoggerOption is an option that configures a Logger.
type LoggerOption interface {
	applyLogger(*loggerConfig)
}

type loggerOptionFunc func(*loggerConfig)

func (f loggerOptionFunc) applyLogger(c *loggerConfig) {
	f(c)
}

// WithLogger returns an option that sets the Kratos logger used by Logger.
//
// If logger is nil, the option leaves the current logger unchanged.
func WithLogger(logger log.Logger) LoggerOption {
	return loggerOptionFunc(func(c *loggerConfig) {
		if logger != nil {
			c.logger = logger
		}
	})
}

// Logger adapts a Kratos logger to cron's Printf-compatible logger shape.
type Logger struct {
	logger log.Logger
}

// NewLogger creates a cron-compatible Logger.
//
// NewLogger uses Kratos' default logger unless opts replace it.
func NewLogger(opts ...LoggerOption) *Logger {
	c := &loggerConfig{
		logger: log.DefaultLogger,
	}
	for _, opt := range opts {
		if opt != nil {
			opt.applyLogger(c)
		}
	}
	return &Logger{logger: c.logger}
}

// Printf writes a formatted message at info level.
//
// Printf discards errors returned by the underlying Kratos logger because
// cron's Printf logger shape has no error return value.
func (l *Logger) Printf(format string, v ...any) {
	_ = l.logger.Log(log.LevelInfo, "msg", fmt.Sprintf(format, v...))
}
