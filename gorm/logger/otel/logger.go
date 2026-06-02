package otel

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/log"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const scopeName = "github.com/go-fries/fries/gorm/logger/otel/v3"

var (
	_ logger.Interface  = (*Logger)(nil)
	_ gorm.ParamsFilter = (*Logger)(nil)
)

// Logger emits GORM logs through the OpenTelemetry Logs API.
type Logger struct {
	logger                    log.Logger
	logAttributes             []log.KeyValue
	logAttributeFuncs         []LogAttributeFunc
	level                     logger.LogLevel
	slowThreshold             time.Duration
	ignoreRecordNotFoundError bool
	parameterizedQueries      bool
}

// New creates a [Logger] that implements [logger.Interface].
func New(opts ...Option) *Logger {
	cfg := newConfig(opts...)

	return &Logger{
		logger:                    cfg.newLogger(scopeName),
		logAttributes:             cfg.logAttributes,
		logAttributeFuncs:         cfg.logAttributeFuncs,
		level:                     cfg.level,
		slowThreshold:             cfg.slowThreshold,
		ignoreRecordNotFoundError: cfg.ignoreRecordNotFoundError,
		parameterizedQueries:      cfg.parameterizedQueries,
	}
}

// LogMode returns a copy of l with the given GORM log level.
func (l *Logger) LogMode(level logger.LogLevel) logger.Interface {
	copied := *l
	copied.level = level
	return &copied
}

// Info emits an informational GORM log record.
func (l *Logger) Info(ctx context.Context, msg string, data ...any) {
	if l.level >= logger.Info && l.enabled(ctx, log.SeverityInfo) {
		l.emit(ctx, log.SeverityInfo, "INFO", fmt.Sprintf(msg, data...), nil)
	}
}

// Warn emits a warning GORM log record.
func (l *Logger) Warn(ctx context.Context, msg string, data ...any) {
	if l.level >= logger.Warn && l.enabled(ctx, log.SeverityWarn) {
		l.emit(ctx, log.SeverityWarn, "WARN", fmt.Sprintf(msg, data...), nil)
	}
}

// Error emits an error GORM log record.
func (l *Logger) Error(ctx context.Context, msg string, data ...any) {
	if l.level >= logger.Error && l.enabled(ctx, log.SeverityError) {
		l.emit(ctx, log.SeverityError, "ERROR", fmt.Sprintf(msg, data...), nil)
	}
}

// Trace emits a GORM SQL trace log record for errors, slow SQL, or info-level
// query logging.
func (l *Logger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.level <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && l.level >= logger.Error &&
		(!errors.Is(err, logger.ErrRecordNotFound) || !l.ignoreRecordNotFoundError):
		if !l.enabled(ctx, log.SeverityError) {
			return
		}
		sql, rows := fc()
		l.emitSQL(ctx, log.SeverityError, "ERROR", "gorm.sql.error", sql, rows, elapsed, err)
	case elapsed > l.slowThreshold && l.slowThreshold != 0 && l.level >= logger.Warn:
		if !l.enabled(ctx, log.SeverityWarn) {
			return
		}
		sql, rows := fc()
		l.emitSQL(ctx, log.SeverityWarn, "WARN", "gorm.sql.slow", sql, rows, elapsed, nil)
	case l.level == logger.Info:
		if !l.enabled(ctx, log.SeverityInfo) {
			return
		}
		sql, rows := fc()
		l.emitSQL(ctx, log.SeverityInfo, "INFO", "gorm.sql", sql, rows, elapsed, nil)
	}
}

// ParamsFilter controls how GORM renders SQL parameters before trace logging.
func (l *Logger) ParamsFilter(_ context.Context, sql string, params ...any) (string, []any) {
	if l.parameterizedQueries {
		return sql, nil
	}
	return sql, params
}

func (l *Logger) emit(ctx context.Context, severity log.Severity, severityText, body string, attrs []log.KeyValue) {
	var record log.Record
	record.SetTimestamp(time.Now())
	record.SetSeverity(severity)
	record.SetSeverityText(severityText)
	record.SetBody(log.StringValue(body))
	record.AddAttributes(attrs...)
	record.AddAttributes(l.logAttributes...)
	for _, fn := range l.logAttributeFuncs {
		record.AddAttributes(fn(ctx)...)
	}

	l.logger.Emit(ctx, record)
}

func (l *Logger) enabled(ctx context.Context, severity log.Severity) bool {
	return l.logger.Enabled(ctx, log.EnabledParameters{Severity: severity})
}

func (l *Logger) emitSQL(
	ctx context.Context,
	severity log.Severity,
	severityText string,
	eventName string,
	sql string,
	rows int64,
	elapsed time.Duration,
	err error,
) {
	attrs := []log.KeyValue{
		log.String(string(semconv.DBQueryTextKey), sql),
		log.Int64("gorm.rows_affected", rows),
		log.Float64("gorm.elapsed_ms", float64(elapsed.Nanoseconds())/1e6),
		log.String("gorm.event", eventName),
	}
	if err != nil {
		errorType := semconv.ErrorType(err)
		attrs = append(attrs, log.String(string(errorType.Key), errorType.Value.AsString()))
		attrs = append(attrs, log.String("error.message", err.Error()))
	}

	l.emit(ctx, severity, severityText, eventName, attrs)
}
