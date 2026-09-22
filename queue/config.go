package queue

import "github.com/go-fries/fries/codec/v4"

type definitionConfig struct {
	codec codec.Codec
}

// DefinitionOption configures a shared task definition.
type DefinitionOption interface {
	applyDefinition(*definitionConfig)
}

type definitionOptionFunc func(*definitionConfig)

func (f definitionOptionFunc) applyDefinition(c *definitionConfig) {
	f(c)
}

// WithCodec sets the codec used by a definition for enqueueing and handling.
// A nil codec selects JSON. If repeated, the last option wins.
// The codec is shared, not cloned; it must support concurrent use when the
// definition is used concurrently by producers or workers.
func WithCodec(codec codec.Codec) DefinitionOption {
	return definitionOptionFunc(func(c *definitionConfig) {
		c.codec = codec
		if codec == nil {
			c.codec = defaultCodec
		}
	})
}

func newDefinitionConfig(opts ...DefinitionOption) *definitionConfig {
	c := &definitionConfig{codec: defaultCodec}
	for _, opt := range opts {
		opt.applyDefinition(c)
	}
	return c
}
