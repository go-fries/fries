package stack

import (
	"errors"

	"github.com/go-kratos/kratos/v2/log"
)

type stackLogger []log.Logger

var _ log.Logger = (stackLogger)(nil)

func New(loggers ...log.Logger) log.Logger {
	return stackLogger(loggers)
}

func (s stackLogger) Log(level log.Level, keyvals ...any) (err error) {
	for _, logger := range s {
		if e := logger.Log(level, keyvals...); e != nil {
			err = errors.Join(err, e)
		}
	}
	return err
}
