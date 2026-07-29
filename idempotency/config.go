package idempotency

import (
	"time"

	codecjson "github.com/go-fries/fries/codec/json/v4"
	"github.com/go-fries/fries/codec/v4"
)

const (
	defaultExecutionTTL        = 30 * time.Second
	defaultResultTTL           = 24 * time.Hour
	defaultFinalizationTimeout = 5 * time.Second
)

type config struct {
	executionTTL        time.Duration
	resultTTL           time.Duration
	finalizationTimeout time.Duration
	codec               codec.Codec
}

// Option configures an [Executor].
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

// WithFinalizationTimeout sets the time available to complete or abort a
// claim. Non-positive values are ignored.
func WithFinalizationTimeout(timeout time.Duration) Option {
	return optionFunc(func(c *config) {
		if timeout > 0 {
			c.finalizationTimeout = timeout
		}
	})
}

// WithCodec sets the codec used to persist values handled by [DoValue]. A nil
// codec leaves the current codec unchanged.
func WithCodec(value codec.Codec) Option {
	return optionFunc(func(c *config) {
		if value != nil {
			c.codec = value
		}
	})
}

func newConfig(options ...Option) config {
	c := config{
		executionTTL:        defaultExecutionTTL,
		resultTTL:           defaultResultTTL,
		finalizationTimeout: defaultFinalizationTimeout,
		codec:               codecjson.Codec{},
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

// ExecuteOption configures one [Executor.Do] or [DoValue] call.
type ExecuteOption interface {
	applyExecute(*executeConfig)
}

type executeOptionFunc func(*executeConfig)

func (f executeOptionFunc) applyExecute(c *executeConfig) {
	f(c)
}

// ExecutionTTLOption configures the execution claim TTL for an [Executor] or
// one [Executor.Do] or [DoValue] call.
type ExecutionTTLOption interface {
	Option
	ExecuteOption
	executionTTLOption()
}

type executionTTLOption struct {
	ttl time.Duration
}

func (o executionTTLOption) apply(c *config) {
	if o.ttl > 0 {
		c.executionTTL = o.ttl
	}
}

func (o executionTTLOption) applyExecute(c *executeConfig) {
	if o.ttl > 0 {
		c.executionTTL = o.ttl
	}
}

func (executionTTLOption) executionTTLOption() {}

// WithExecutionTTL sets the execution claim TTL. When passed to [New] it
// changes the [Executor] default; when passed to [Executor.Do] or [DoValue] it
// overrides that default for the current call. Non-positive values are
// ignored.
func WithExecutionTTL(ttl time.Duration) ExecutionTTLOption {
	return executionTTLOption{ttl: ttl}
}

// ResultTTLOption configures the completed record TTL for an [Executor] or one
// [Executor.Do] or [DoValue] call.
type ResultTTLOption interface {
	Option
	ExecuteOption
	resultTTLOption()
}

type resultTTLOption struct {
	ttl time.Duration
}

func (o resultTTLOption) apply(c *config) {
	if o.ttl > 0 {
		c.resultTTL = o.ttl
	}
}

func (o resultTTLOption) applyExecute(c *executeConfig) {
	if o.ttl > 0 {
		c.resultTTL = o.ttl
	}
}

func (resultTTLOption) resultTTLOption() {}

// WithResultTTL sets the completed record TTL. When passed to [New] it changes
// the [Executor] default; when passed to [Executor.Do] or [DoValue] it
// overrides that default for the current call. Non-positive values are
// ignored.
func WithResultTTL(ttl time.Duration) ResultTTLOption {
	return resultTTLOption{ttl: ttl}
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
