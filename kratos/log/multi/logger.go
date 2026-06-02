package multi // import "github.com/go-fries/fries/kratos/log/multi/v3"

import (
	"errors"

	"github.com/go-kratos/kratos/v2/log"
)

var _ log.Logger = (*Logger)(nil)

// Logger dispatches each Kratos log call to multiple [log.Logger]
// implementations.
type Logger struct {
	loggers []log.Logger
}

// New creates a [Logger] that dispatches Kratos logs to each supplied logger.
func New(loggers ...log.Logger) *Logger {
	l := &Logger{}
	for _, item := range loggers {
		if item != nil {
			l.loggers = append(l.loggers, item)
		}
	}
	return l
}

// Log dispatches a Kratos log call to each underlying logger.
func (l *Logger) Log(level log.Level, keyvals ...any) error {
	switch len(l.loggers) {
	case 0:
		return nil
	case 1:
		return l.loggers[0].Log(level, keyvals...)
	}

	var errs []error
	for _, item := range l.loggers {
		if err := item.Log(level, keyvals...); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
