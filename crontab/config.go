package crontab

import "log/slog"

type loggerConfig struct {
	logger *slog.Logger
}

type serverConfig struct {
	logger *slog.Logger
}

// ServerOption is an option that configures a Server.
type ServerOption interface {
	applyServer(*serverConfig)
}

// LoggerOption is an option that configures a Logger.
type LoggerOption interface {
	applyLogger(*loggerConfig)
}

// LoggerServerOption is an option that configures both Logger and Server.
type LoggerServerOption interface {
	LoggerOption
	ServerOption
}

type loggerOption struct {
	logger *slog.Logger
}

func (o loggerOption) applyLogger(c *loggerConfig) {
	if o.logger != nil {
		c.logger = o.logger
	}
}

func (o loggerOption) applyServer(c *serverConfig) {
	if o.logger != nil {
		c.logger = o.logger
	}
}

// WithLogger returns an option that sets the slog logger used by Logger or Server.
//
// If logger is nil, the option leaves the current logger unchanged.
func WithLogger(logger *slog.Logger) LoggerServerOption {
	return loggerOption{logger: logger}
}
