package stack

import (
	"errors"

	"github.com/go-kratos/kratos/v2/log"
)

type stackLogger []log.Logger

var _ log.Logger = stackLogger(nil)

func New(loggers ...log.Logger) log.Logger {
	return stackLogger(loggers)
}

func (s stackLogger) Log(level log.Level, keyvals ...any) error {
	var errs []error
	for _, logger := range s {
		if err := logger.Log(level, keyvals...); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
