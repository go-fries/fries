package ctxvaluer

type field struct {
	key   string
	value Valuer
}

type config struct {
	fields       []field
	currentGroup bool
}

// Option configures a Handler when it is constructed.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

// WithValue registers a dynamic attribute. Registrations are evaluated in order,
// including registrations with duplicate keys. A nil value causes a panic when
// NewHandler applies this option.
func WithValue(key string, value Valuer) Option {
	return optionFunc(func(c *config) {
		if value == nil {
			panic("ctxvaluer: nil valuer")
		}
		c.fields = append(c.fields, field{key: key, value: value})
	})
}

// WithCurrentGroup makes dynamic attributes follow the wrapped handler's current
// group. In this mode, WithAttrs and WithGroup delegate directly to that handler.
// Without this option, groups created on the wrapper do not contain dynamic
// attributes, and grouped WithAttrs are retained until Handle.
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
