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

// WithLogger sets the Kratos logger used by Logger.
func WithLogger(logger log.Logger) LoggerOption {
	return loggerOptionFunc(func(c *loggerConfig) {
		if logger != nil {
			c.logger = logger
		}
	})
}

// Logger adapts a Kratos logger to cron's Printf logger shape.
type Logger struct {
	logger log.Logger
}

// NewLogger creates a cron-compatible logger with opts.
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

// Printf writes a formatted info-level log message.
func (l *Logger) Printf(format string, v ...any) {
	_ = l.logger.Log(log.LevelInfo, "msg", fmt.Sprintf(format, v...))
}
