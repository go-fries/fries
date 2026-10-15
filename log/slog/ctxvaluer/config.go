package ctxvaluer

type field struct {
	key   string
	value Valuer
}

type config struct {
	fields       []field
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
// [Handler.Handle] evaluates registered [Valuer] functions in registration order,
// including registrations with duplicate keys. [NewHandler] panics if value is nil.
func WithValue(key string, value Valuer) Option {
	return optionFunc(func(c *config) {
		if value == nil {
			panic("ctxvaluer: nil valuer")
		}
		c.fields = append(c.fields, field{key: key, value: value})
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
