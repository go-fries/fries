package multi // import "github.com/go-fries/fries/gorm/logger/multi/v4"

import (
	"context"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	_ logger.Interface  = (*Logger)(nil)
	_ gorm.ParamsFilter = (*Logger)(nil)
)

// Logger dispatches each GORM log call to multiple [logger.Interface]
// implementations.
type Logger struct {
	loggers []logger.Interface
}

// New creates a [Logger] that dispatches GORM logs to each supplied logger.
func New(loggers ...logger.Interface) *Logger {
	l := &Logger{}
	for _, item := range loggers {
		if item != nil {
			l.loggers = append(l.loggers, item)
		}
	}
	return l
}

// LogMode returns a new [logger.Interface] with the log level applied to each
// underlying logger.
func (l *Logger) LogMode(level logger.LogLevel) logger.Interface {
	loggers := make([]logger.Interface, 0, len(l.loggers))
	for _, item := range l.loggers {
		next := item.LogMode(level)
		if next != nil {
			loggers = append(loggers, next)
		}
	}
	return New(loggers...)
}

// Info dispatches an informational GORM log call to each underlying logger.
func (l *Logger) Info(ctx context.Context, msg string, data ...any) {
	for _, item := range l.loggers {
		item.Info(ctx, msg, data...)
	}
}

// Warn dispatches a warning GORM log call to each underlying logger.
func (l *Logger) Warn(ctx context.Context, msg string, data ...any) {
	for _, item := range l.loggers {
		item.Warn(ctx, msg, data...)
	}
}

// Error dispatches an error GORM log call to each underlying logger.
func (l *Logger) Error(ctx context.Context, msg string, data ...any) {
	for _, item := range l.loggers {
		item.Error(ctx, msg, data...)
	}
}

// Trace dispatches a SQL trace log call to each underlying logger.
func (l *Logger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	fc = safeTraceFunc(fc)

	switch len(l.loggers) {
	case 0:
		return
	case 1:
		l.loggers[0].Trace(ctx, begin, fc, err)
		return
	}

	cached := cacheTraceFunc(fc)
	for _, item := range l.loggers {
		item.Trace(ctx, begin, cached, err)
	}
}

// ParamsFilter applies every underlying [gorm.ParamsFilter] in order.
func (l *Logger) ParamsFilter(ctx context.Context, sql string, params ...any) (string, []any) {
	nextSQL, nextParams := sql, params
	for _, item := range l.loggers {
		filter, ok := item.(gorm.ParamsFilter)
		if !ok {
			continue
		}
		nextSQL, nextParams = filter.ParamsFilter(ctx, nextSQL, nextParams...)
	}
	return nextSQL, nextParams
}

func safeTraceFunc(fc func() (sql string, rowsAffected int64)) func() (sql string, rowsAffected int64) {
	if fc != nil {
		return fc
	}

	return func() (string, int64) {
		return "", 0
	}
}

func cacheTraceFunc(fc func() (sql string, rowsAffected int64)) func() (sql string, rowsAffected int64) {
	var (
		once sync.Once
		sql  string
		rows int64
	)

	return func() (string, int64) {
		once.Do(func() {
			sql, rows = fc()
		})
		return sql, rows
	}
}
