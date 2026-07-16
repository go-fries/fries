package event

type config struct {
	middleware []Middleware
}

// Option configures a [Dispatcher].
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

// WithMiddleware appends immutable [Dispatcher] middleware. Middleware is
// applied from outermost to innermost in declaration order. Nil middleware is
// ignored.
func WithMiddleware(middleware ...Middleware) Option {
	return optionFunc(func(c *config) {
		for _, item := range middleware {
			if item != nil {
				c.middleware = append(c.middleware, item)
			}
		}
	})
}

func newConfig(options ...Option) config {
	var c config
	for _, option := range options {
		if option != nil {
			option.apply(&c)
		}
	}
	return c
}

type dispatchConfig struct {
	concurrency     int
	continueOnError bool
}

// DispatchOption configures one [Dispatcher.Dispatch] call.
type DispatchOption interface {
	applyDispatch(*dispatchConfig)
}

type dispatchOptionFunc func(*dispatchConfig)

func (f dispatchOptionFunc) applyDispatch(c *dispatchConfig) {
	f(c)
}

// WithConcurrency bounds the number of handlers active during one
// [Dispatcher.Dispatch].
// Non-positive limits are ignored and preserve serial execution.
func WithConcurrency(limit int) DispatchOption {
	return dispatchOptionFunc(func(c *dispatchConfig) {
		if limit > 0 {
			c.concurrency = limit
		}
	})
}

// ContinueOnError runs the remaining handlers and joins their errors in
// registration order.
func ContinueOnError() DispatchOption {
	return dispatchOptionFunc(func(c *dispatchConfig) {
		c.continueOnError = true
	})
}

func newDispatchConfig(options ...DispatchOption) dispatchConfig {
	c := dispatchConfig{concurrency: 1}
	for _, option := range options {
		if option != nil {
			option.applyDispatch(&c)
		}
	}
	return c
}
