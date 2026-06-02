package otel

import (
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"gorm.io/gorm/logger"
)

type config struct {
	provider                  log.LoggerProvider
	version                   string
	schemaURL                 string
	attributes                []attribute.KeyValue
	level                     logger.LogLevel
	slowThreshold             time.Duration
	ignoreRecordNotFoundError bool
	parameterizedQueries      bool
}

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
