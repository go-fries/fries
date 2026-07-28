package idempotency

import "time"

const (
	defaultExecutionTTL        = 30 * time.Second
	defaultResultTTL           = 24 * time.Hour
	defaultFinalizationTimeout = 5 * time.Second
)

type config struct {
	executionTTL        time.Duration
	resultTTL           time.Duration
	finalizationTimeout time.Duration
}

// Option configures an Executor.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

// WithDefaultExecutionTTL sets the execution claim TTL used when a call does
// not provide WithExecutionTTL. Non-positive values are ignored.
func WithDefaultExecutionTTL(ttl time.Duration) Option {
	return optionFunc(func(c *config) {
		if ttl > 0 {
			c.executionTTL = ttl
		}
	})
}

// WithDefaultResultTTL sets the completed record TTL used when a call does not
// provide WithResultTTL. Non-positive values are ignored.
func WithDefaultResultTTL(ttl time.Duration) Option {
	return optionFunc(func(c *config) {
		if ttl > 0 {
			c.resultTTL = ttl
		}
	})
}

// WithFinalizationTimeout sets the time available to complete or abort a
// claim after Handler execution. Non-positive values are ignored.
func WithFinalizationTimeout(timeout time.Duration) Option {
	return optionFunc(func(c *config) {
		if timeout > 0 {
			c.finalizationTimeout = timeout
		}
	})
}

func newConfig(options ...Option) config {
	c := config{
		executionTTL:        defaultExecutionTTL,
		resultTTL:           defaultResultTTL,
		finalizationTimeout: defaultFinalizationTimeout,
	}
	for _, option := range options {
		if option != nil {
			option.apply(&c)
		}
	}
	return c
}

type executeConfig struct {
	executionTTL time.Duration
	resultTTL    time.Duration
	fingerprint  string
}

// ExecuteOption configures one Executor.Do call.
type ExecuteOption interface {
	applyExecute(*executeConfig)
}

type executeOptionFunc func(*executeConfig)

func (f executeOptionFunc) applyExecute(c *executeConfig) {
	f(c)
}

// WithExecutionTTL overrides the execution claim TTL for one call.
// Non-positive values are ignored.
func WithExecutionTTL(ttl time.Duration) ExecuteOption {
	return executeOptionFunc(func(c *executeConfig) {
		if ttl > 0 {
			c.executionTTL = ttl
		}
	})
}

// WithResultTTL overrides the completed record TTL for one call.
// Non-positive values are ignored.
func WithResultTTL(ttl time.Duration) ExecuteOption {
	return executeOptionFunc(func(c *executeConfig) {
		if ttl > 0 {
			c.resultTTL = ttl
		}
	})
}

// WithFingerprint associates the key with stable input identity. An empty
// fingerprint disables input conflict detection.
func WithFingerprint(fingerprint string) ExecuteOption {
	return executeOptionFunc(func(c *executeConfig) {
		c.fingerprint = fingerprint
	})
}

func newExecuteConfig(c config, options ...ExecuteOption) executeConfig {
	execution := executeConfig{
		executionTTL: c.executionTTL,
		resultTTL:    c.resultTTL,
	}
	for _, option := range options {
		if option != nil {
			option.applyExecute(&execution)
		}
	}
	return execution
}
