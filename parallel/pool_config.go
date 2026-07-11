package parallel

type poolConfig struct {
	queueSize int
}

// PoolOption configures a Pool.
type PoolOption interface {
	apply(*poolConfig)
}

type poolOptionFunc func(*poolConfig)

func (f poolOptionFunc) apply(config *poolConfig) {
	f(config)
}

// WithQueueSize configures the number of tasks that may wait for a worker.
//
// A size of zero creates an unbuffered queue. Negative values keep the default
// queue size, which equals the worker count.
func WithQueueSize(size int) PoolOption {
	return poolOptionFunc(func(config *poolConfig) {
		if size >= 0 {
			config.queueSize = size
		}
	})
}

func newPoolConfig(workers int, options ...PoolOption) poolConfig {
	config := poolConfig{queueSize: workers}
	for _, option := range options {
		option.apply(&config)
	}

	return config
}
