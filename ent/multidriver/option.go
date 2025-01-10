package multidriver

import "entgo.io/ent/dialect"

type Option func(*Driver)

func WithWriter(w dialect.Driver) Option {
	return func(d *Driver) {
		d.writer = w
	}
}

func WithReaders(r ...dialect.Driver) Option {
	return func(d *Driver) {
		d.readers = r
	}
}

func WithPolicy(p Policy) Option {
	return func(d *Driver) {
		d.policy = p
	}
}
