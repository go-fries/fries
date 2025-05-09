package redis

import "strings"

type Option interface {
	apply(d *Driver)
}

type optionFunc func(d *Driver)

func (f optionFunc) apply(d *Driver) {
	f(d)
}

func WithPrefix(prefix string) Option {
	return optionFunc(func(d *Driver) {
		if prefix != "" {
			d.prefix = strings.TrimSuffix(prefix, ":") + ":"
		}
	})
}
