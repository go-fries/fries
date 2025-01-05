package retry

import (
	"errors"

	"github.com/go-fires/fires/v3/hyperf/jet"
	"github.com/go-fires/fires/v3/hyperf/jet/middleware/timeout"
)

var DefaultAllow AllowFunc = func(err error) bool {
	return jet.IsHTTPTransporterServerError(err) &&
		errors.Is(err, timeout.ErrTimeout)
}

type AllowFunc func(err error) bool

func OrAllowFuncs(fs ...AllowFunc) AllowFunc {
	return func(err error) bool {
		for _, f := range fs {
			if f(err) {
				return true
			}
		}
		return false
	}
}

type options struct {
	attempts int
	backoff  BackoffFunc
	allow    AllowFunc // allow retry
}

type Option func(o *options)

func Attempts(attempts int) Option {
	return func(o *options) {
		o.attempts = attempts
	}
}

func Allow(f AllowFunc) Option {
	return func(o *options) {
		o.allow = f
	}
}

func Backoff(f BackoffFunc) Option {
	return func(o *options) {
		o.backoff = f
	}
}
