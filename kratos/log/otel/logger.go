package otel // import "github.com/go-fries/fries/kratos/log/otel/v3"

import (
	"context"
	"fmt"
	"time"

	kratoslog "github.com/go-kratos/kratos/v2/log"
	"go.opentelemetry.io/otel/log"
)

type Logger struct {
	logger log.Logger
}

const scopeName = "github.com/go-fries/fries/kratos/log/otel/v3"

var _ kratoslog.Logger = (*Logger)(nil)

func NewLogger(opts ...Option) *Logger {
	cfg := newConfig(opts...)

	logger := cfg.provider.Logger(
		scopeName,
		log.WithInstrumentationVersion(Version()),
	)

	return &Logger{
		logger: logger,
	}
}

func (l *Logger) Log(level kratoslog.Level, keyvals ...any) error {
	severity := convertLevel(level)
	ctx := contextFromKVs(context.Background(), keyvals...)
	if !l.logger.Enabled(ctx, log.EnabledParameters{Severity: severity}) {
		return nil
	}

	var record log.Record
	record.SetTimestamp(time.Now())
	record.SetSeverity(severity)
	record.SetSeverityText(level.String())

	ctx, body, kvs := convertKVs(ctx, keyvals...)
	if body != "" {
		record.SetBody(log.StringValue(body))
	}

	record.AddAttributes(kvs...)

	l.logger.Emit(ctx, record)
	return nil
}

func contextFromKVs(ctx context.Context, keyvals ...any) context.Context {
	for i := 1; i < len(keyvals); i += 2 {
		if vCtx, ok := keyvals[i].(context.Context); ok {
			ctx = vCtx
		}
	}
	return ctx
}

func convertLevel(level kratoslog.Level) log.Severity {
	switch level {
	case kratoslog.LevelDebug:
		return log.SeverityDebug
	case kratoslog.LevelInfo:
		return log.SeverityInfo
	case kratoslog.LevelWarn:
		return log.SeverityWarn
	case kratoslog.LevelError:
		return log.SeverityError
	case kratoslog.LevelFatal:
		return log.SeverityFatal1
	default:
		return log.SeverityUndefined
	}
}

// convertValue converts a value to an OpenTelemetry log.Value.
func convertKVs(ctx context.Context, keyvals ...any) (context.Context, string, []log.KeyValue) {
	if len(keyvals)%2 != 0 {
		keyvals = append(keyvals, nil)
	}

	body := ""
	kvs := make([]log.KeyValue, 0, len(keyvals)/2) //nolint:mnd
	for i := 0; i < len(keyvals); i += 2 {
		key, ok := keyvals[i].(string)
		if !ok {
			key = fmt.Sprintf("%+v", keyvals[i])
		}

		if key == kratoslog.DefaultMessageKey {
			body = fmt.Sprintf("%+v", keyvals[i+1])
			continue
		}

		v := keyvals[i+1]
		if vCtx, ok := v.(context.Context); ok {
			// Special case when a field is of context.Context type.
			ctx = vCtx
			continue
		}

		kvs = append(kvs, log.KeyValue{
			Key:   key,
			Value: convertValue(keyvals[i+1]),
		})
	}

	return ctx, body, kvs
}
