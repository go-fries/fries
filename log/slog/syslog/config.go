//go:build !windows && !plan9

package syslog

import (
	"log/slog"
	stdsyslog "log/syslog"
)

type config struct {
	level    slog.Leveler
	priority stdsyslog.Priority
	tag      string
}

func newConfig(opts ...Option) *config {
	cfg := &config{
		priority: stdsyslog.LOG_USER,
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt.apply(cfg)
	}
	return cfg
}

// Option configures a [Handler].
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

// WithLevel sets the minimum slog level handled by [Handler].
func WithLevel(level slog.Leveler) Option {
	return optionFunc(func(c *config) {
		c.level = level
	})
}

// WithPriority sets the syslog facility used by [Dial].
func WithPriority(priority stdsyslog.Priority) Option {
	return optionFunc(func(c *config) {
		c.priority = priority
	})
}

// WithTag sets the syslog tag used by [Dial].
func WithTag(tag string) Option {
	return optionFunc(func(c *config) {
		c.tag = tag
	})
}
