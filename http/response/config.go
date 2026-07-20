package response

type config struct {
	code *int
	data any
}

// Option configures a response created by [Success] or [Failure].
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

// WithCode sets the optional application code.
//
// The application code is independent of the HTTP status code passed to
// [Write].
func WithCode(code int) Option {
	return optionFunc(func(c *config) {
		value := code
		c.code = &value
	})
}

// WithData sets the response data.
//
// It is most useful for attaching safe, structured details to a response
// created by [Failure]. When used with [Success], it replaces the data passed
// to Success.
func WithData(data any) Option {
	return optionFunc(func(c *config) {
		c.data = data
	})
}

func newConfig(data any, options ...Option) config {
	c := config{data: data}
	for _, option := range options {
		if option != nil {
			option.apply(&c)
		}
	}
	return c
}
