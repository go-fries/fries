package parallel

import "fmt"

type poolConfig struct {
	queueSize int
}

// PoolOption configures a Pool.
type PoolOption interface {
	apply(*poolConfig) error
}

type poolOptionFunc func(*poolConfig) error

func (f poolOptionFunc) apply(config *poolConfig) error {
	return f(config)
}

// WithQueueSize configures the number of tasks that may wait for a worker.
//
// A size of zero creates an unbuffered queue, so Submit waits until a worker
// can accept the task. The default queue size equals the worker count.
func WithQueueSize(size int) PoolOption {
	return poolOptionFunc(func(config *poolConfig) error {
		if size < 0 {
			return fmt.Errorf("%w: got %d", ErrInvalidQueueSize, size)
		}

		config.queueSize = size

		return nil
	})
}

func newPoolConfig(workers int, options ...PoolOption) (poolConfig, error) {
	config := poolConfig{queueSize: workers}
	for _, option := range options {
		if err := option.apply(&config); err != nil {
			return poolConfig{}, err
		}
	}

	return config, nil
}
