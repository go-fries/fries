package recovery

const defaultStackSize = 4 << 10

type config struct {
	stackSize int
}

// Option configures recovery middleware.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

// WithStackSize sets the stack buffer size in bytes. Non-positive sizes are
// ignored.
func WithStackSize(size int) Option {
	return optionFunc(func(c *config) {
		if size > 0 {
			c.stackSize = size
		}
	})
}

func newConfig(options ...Option) config {
	c := config{stackSize: defaultStackSize}
	for _, option := range options {
		if option != nil {
			option.apply(&c)
		}
	}
	return c
}
