package otel

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm/logger"
)

type config struct {
	provider                  log.LoggerProvider
	version                   string
	schemaURL                 string
	attributes                []attribute.KeyValue
	logAttributes             []log.KeyValue
	logAttributeFuncs         []LogAttributeFunc
	level                     logger.LogLevel
	slowThreshold             time.Duration
	ignoreRecordNotFoundError bool
	parameterizedQueries      bool
}

// LogAttributeFunc returns OpenTelemetry log record attributes for ctx.
type LogAttributeFunc func(ctx context.Context) []log.KeyValue

// Option configures a [Logger].
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

// WithLoggerProvider sets the [log.LoggerProvider] used to create the
// underlying [log.Logger].
func WithLoggerProvider(provider log.LoggerProvider) Option {
	return optionFunc(func(c *config) {
		if provider != nil {
			c.provider = provider
		}
	})
}

// WithVersion sets the instrumentation scope version reported to OpenTelemetry.
func WithVersion(version string) Option {
	return optionFunc(func(c *config) {
		if version != "" {
			c.version = version
		}
	})
}

// WithSchemaURL sets the schema URL reported to OpenTelemetry for log records
// emitted by the instrumentation scope.
func WithSchemaURL(schemaURL string) Option {
	return optionFunc(func(c *config) {
		if schemaURL != "" {
			c.schemaURL = schemaURL
		}
	})
}

// WithAttributes adds instrumentation scope [attribute.KeyValue] attributes
// reported to OpenTelemetry.
func WithAttributes(attributes ...attribute.KeyValue) Option {
	return optionFunc(func(c *config) {
		c.attributes = append(c.attributes, attributes...)
	})
}

// WithLogAttributes adds OpenTelemetry log record attributes emitted with each
// log record.
func WithLogAttributes(attributes ...log.KeyValue) Option {
	return optionFunc(func(c *config) {
		c.logAttributes = append(c.logAttributes, attributes...)
	})
}

// WithLogAttributeFuncs adds functions that return OpenTelemetry log record
// attributes for each emitted log record.
func WithLogAttributeFuncs(funcs ...LogAttributeFunc) Option {
	return optionFunc(func(c *config) {
		for _, fn := range funcs {
			if fn != nil {
				c.logAttributeFuncs = append(c.logAttributeFuncs, fn)
			}
		}
	})
}

// WithTraceContext adds the current span trace and span IDs to each emitted log
// record when they are available in the log context.
func WithTraceContext() Option {
	return WithLogAttributeFuncs(func(ctx context.Context) []log.KeyValue {
		span := trace.SpanContextFromContext(ctx)
		if !span.HasTraceID() && !span.HasSpanID() {
			return nil
		}

		attrs := make([]log.KeyValue, 0, 2) //nolint:mnd
		if span.HasTraceID() {
			attrs = append(attrs, log.String("trace.id", span.TraceID().String()))
		}
		if span.HasSpanID() {
			attrs = append(attrs, log.String("span.id", span.SpanID().String()))
		}
		return attrs
	})
}

// WithLogLevel sets the GORM log level used by [Logger].
func WithLogLevel(level logger.LogLevel) Option {
	return optionFunc(func(c *config) {
		c.level = level
	})
}

// WithSlowThreshold sets the duration after which GORM trace logs are emitted
// as slow SQL warnings. A zero duration disables slow SQL warnings.
func WithSlowThreshold(threshold time.Duration) Option {
	return optionFunc(func(c *config) {
		c.slowThreshold = threshold
	})
}

// WithIgnoreRecordNotFoundError controls whether [logger.ErrRecordNotFound]
// is emitted as an error log record.
func WithIgnoreRecordNotFoundError(ignore bool) Option {
	return optionFunc(func(c *config) {
		c.ignoreRecordNotFoundError = ignore
	})
}

// WithParameterizedQueries controls whether SQL parameters are omitted from
// rendered GORM SQL logs.
func WithParameterizedQueries(parameterized bool) Option {
	return optionFunc(func(c *config) {
		c.parameterizedQueries = parameterized
	})
}

func newConfig(opts ...Option) *config {
	cfg := &config{
		provider:                  global.GetLoggerProvider(),
		version:                   Version(),
		level:                     logger.Warn,
		slowThreshold:             200 * time.Millisecond,
		ignoreRecordNotFoundError: true,
	}
	for _, opt := range opts {
		opt.apply(cfg)
	}
	return cfg
}

func (c *config) newLogger(name string) log.Logger {
	opts := []log.LoggerOption{
		log.WithInstrumentationVersion(c.version),
	}
	if c.schemaURL != "" {
		opts = append(opts, log.WithSchemaURL(c.schemaURL))
	}
	if len(c.attributes) > 0 {
		opts = append(opts, log.WithInstrumentationAttributes(c.attributes...))
	}

	return c.provider.Logger(name, opts...)
}
