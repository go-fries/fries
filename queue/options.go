package queue

import "github.com/go-fries/fries/codec/v3"

type Option interface {
	apply(d *Queue)
}

type optionFunc func(d *Queue)

func (f optionFunc) apply(d *Queue) {
	f(d)
}

func WithCodec(codec codec.Codec) Option {
	return optionFunc(func(d *Queue) {
		if codec != nil {
			d.codec = codec
		}
	})
}
