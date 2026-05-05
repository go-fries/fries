//go:build !windows && !plan9

package syslog

import (
	"bytes"
	"fmt"
	"log/syslog"

	"github.com/go-kratos/kratos/v2/log"
)

type Config struct {
	Network string
	Addr    string
	Tag     string
}

type Logger struct {
	config *Config
	conn   *syslog.Writer
}

var _ log.Logger = (*Logger)(nil)

func New(config *Config) *Logger {
	return &Logger{
		config: config,
	}
}

func (l *Logger) Log(level log.Level, keyvals ...any) error {
	if l.conn == nil {
		if err := l.connect(); err != nil {
			return err
		}
	}

	if len(keyvals) == 0 {
		return nil
	}

	if (len(keyvals) & 1) == 1 {
		keyvals = append(keyvals, "KEYVALS UNPAIRED")
	}

	var buf bytes.Buffer

	buf.WriteString(level.String())
	for i := 0; i < len(keyvals); i += 2 {
		_, _ = fmt.Fprintf(&buf, " %s=%v", keyvals[i], keyvals[i+1])
	}

	switch level {
	case log.LevelDebug:
		return l.conn.Debug(buf.String())
	case log.LevelInfo:
		return l.conn.Info(buf.String())
	case log.LevelWarn:
		return l.conn.Warning(buf.String())
	case log.LevelError:
		return l.conn.Err(buf.String())
	case log.LevelFatal:
		return l.conn.Crit(buf.String())
	default:
		return l.conn.Debug(buf.String())
	}
}

func (l *Logger) connect() error {
	var err error
	l.conn, err = syslog.Dial(l.config.Network, l.config.Addr, syslog.LOG_USER, l.config.Tag)
	return err
}

func (l *Logger) Close() error {
	if l.conn == nil {
		return nil
	}

	return l.conn.Close()
}
