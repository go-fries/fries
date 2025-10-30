package redis

import (
	"github.com/go-fries/fries/codec/json/v3"
	"github.com/go-fries/fries/codec/v3"
	"github.com/redis/go-redis/v9"
)

type config struct {
	name  string
	codec codec.Codec
}

type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(o *config) {
	f(o)
}

func WithName(name string) Option {
	return optionFunc(func(c *config) {
		c.name = name
	})
}

func WithCodec(codec codec.Codec) Option {
	return optionFunc(func(c *config) {
		c.codec = codec
	})
}

func newConfig(opts ...Option) *config {
	c := &config{
		name:  "fries:async:queue",
		codec: json.Codec,
	}
	for _, o := range opts {
		o.apply(c)
	}
	return c
}
