package stack

import "github.com/go-kratos/kratos/v2/log"

var _ log.Logger = (*stackLogger)(nil)

type stackLogger struct {
	loggers   []log.Logger
	ignoreErr bool
}

type Option func(*stackLogger)

func IgnoreErr() Option {
	return func(logger *stackLogger) {
		logger.ignoreErr = true
	}
}

func New(loggers []log.Logger, opts ...Option) log.Logger {
	logger := &stackLogger{
		loggers: loggers,
	}

	for _, opt := range opts {
		opt(logger)
	}

	return logger
}

func (s *stackLogger) Log(level log.Level, keyvals ...any) error {
	for _, logger := range s.loggers {
		if err := logger.Log(level, keyvals...); err != nil {
			if !s.ignoreErr {
				return err
			}
		}
	}

	return nil
}
