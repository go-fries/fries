package s3

type config struct {
	root string
}

// Option configures an S3 filesystem.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(cfg *config) {
	f(cfg)
}

// WithRoot stores logical paths below root in the bucket.
func WithRoot(root string) Option {
	return optionFunc(func(cfg *config) {
		cfg.root = root
	})
}

func newConfig(opts ...Option) *config {
	cfg := &config{}
	for _, opt := range opts {
		opt.apply(cfg)
	}
	return cfg
}
