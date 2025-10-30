package channel

const defaultSize = 100

type config struct {
	size int
}

type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(o *config) {
	f(o)
}

func WithSize(size int) Option {
	return optionFunc(func(c *config) {
		c.size = size
	})
}

func newConfig(opts ...Option) *config {
	c := &config{
		size: defaultSize,
	}
	for _, o := range opts {
		o.apply(c)
	}
	return c
}
