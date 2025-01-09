package log

import (
	"github.com/go-kratos/kratos/v2/log"
)

type Manager struct {
	log.Logger
	loggers map[string]log.Logger
}

func NewManager(logger log.Logger) *Manager {
	return &Manager{
		Logger:  logger,
		loggers: make(map[string]log.Logger),
	}
}

func (l *Manager) Register(name string, logger log.Logger) {
	l.loggers[name] = logger
}

func (l *Manager) Channel(names ...string) log.Logger {
	var name string
	if len(names) > 0 {
		name = names[0]
	}

	if name == "" {
		return l
	}

	if logger, ok := l.loggers[name]; ok {
		return logger
	}

	panic("log/manager: the logger [" + name + "] is not registered")
}
