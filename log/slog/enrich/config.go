package enrich

import "slices"

type source struct {
	key     string
	value   Valuer
	extract Extractor
}

type config struct {
	sources      []source
	currentGroup bool
}

// Option configures a [Handler] created by [NewHandler].
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

// WithValue returns an [Option] that registers value under key.
//
// [Handler.Handle] evaluates registered [Valuer] and [Extractor] functions in
// registration order, including registrations with duplicate keys.
// [NewHandler] panics if value is nil.
func WithValue(key string, value Valuer) Option {
	return optionFunc(func(c *config) {
		if value != nil {
			c.sources = append(c.sources, source{key: key, value: value})
		}
	})
}

// WithExtractors returns an [Option] that registers extractors in argument order.
//
// Registrations are interleaved with [WithValue] registrations in option order.
// The order of attributes returned by each [Extractor] is preserved.
// WithExtractors copies the extractors slice; passing no extractors has no effect.
// [NewHandler] panics if any extractor is nil.
func WithExtractors(extractors ...Extractor) Option {
	extractors = slices.Clone(extractors)
	return optionFunc(func(c *config) {
		for _, extract := range extractors {
			if extract != nil {
				c.sources = append(c.sources, source{extract: extract})
			}
		}
	})
}

// WithCurrentGroup returns an [Option] that places dynamic attributes in the
// wrapped handler's current group.
//
// With this option, [Handler.WithAttrs] and [Handler.WithGroup] delegate to the
// wrapped handler. Without this option, [Handler] keeps dynamic attributes outside
// wrapper-owned groups and retains grouped attributes until [Handler.Handle].
func WithCurrentGroup() Option {
	return optionFunc(func(c *config) {
		c.currentGroup = true
	})
}

func newConfig(opts ...Option) *config {
	c := &config{}
	for _, opt := range opts {
		opt.apply(c)
	}
	return c
}
