package redispositioner

import (
	"strings"

	"github.com/go-fries/fries/codec/v3"
)

type BaseAndBufferedOption interface {
	Option
	BufferedOption
}

type prefixOption struct {
	prefix string
}

var _ BaseAndBufferedOption = (*prefixOption)(nil)

func WithPrefix(prefix string) BaseAndBufferedOption {
	return &prefixOption{
		prefix: prefix,
	}
}

func (p *prefixOption) apply(positioner *Positioner) {
	prefix := strings.TrimSuffix(p.prefix, ":")
	if prefix != "" {
		positioner.prefix = prefix + ":"
	}

}

func (p *prefixOption) applyBuffered(positioner *BufferedPositioner) {
	p.apply(positioner.Positioner)
}

type codecOption struct {
	codec codec.Codec
}

var _ BaseAndBufferedOption = (*codecOption)(nil)

func WithCodec(codec codec.Codec) BaseAndBufferedOption {
	return &codecOption{
		codec: codec,
	}
}

func (c *codecOption) apply(positioner *Positioner) {
	positioner.codec = c.codec
}

func (c *codecOption) applyBuffered(positioner *BufferedPositioner) {
	positioner.Positioner.codec = c.codec
}
