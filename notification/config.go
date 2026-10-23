package notification

import (
	"fmt"
	"strings"

	"github.com/go-fries/fries/notification/v4/internal/nilvalue"
)

type config struct {
	channels map[string]Channel
}

// Option configures a [Manager].
type Option interface {
	apply(*config) error
}

type optionFunc func(*config) error

func (f optionFunc) apply(c *config) error {
	return f(c)
}

// WithChannel registers a named channel. Empty names, nil channels (including
// typed nils), and duplicate names return ErrInvalidConfig from New.
// The Manager retains the channel but does not own or close its resources.
func WithChannel(name string, channel Channel) Option {
	return optionFunc(func(c *config) error {
		if strings.TrimSpace(name) == "" || nilvalue.IsNil(channel) {
			return fmt.Errorf("%w: channel %q needs a name and implementation", ErrInvalidConfig, name)
		}
		if _, exists := c.channels[name]; exists {
			return fmt.Errorf("%w: duplicate channel %q", ErrInvalidConfig, name)
		}
		c.channels[name] = channel
		return nil
	})
}

func newConfig(options ...Option) (config, error) {
	c := config{channels: make(map[string]Channel)}
	for _, option := range options {
		if option != nil {
			if err := option.apply(&c); err != nil {
				return config{}, err
			}
		}
	}
	return c, nil
}

type sendConfig struct {
	id string
}

// SendOption configures one [Manager.Send] operation.
type SendOption interface {
	applySend(*sendConfig) error
}

type sendOptionFunc func(*sendConfig) error

func (f sendOptionFunc) applySend(c *sendConfig) error {
	return f(c)
}

// WithID supplies a stable ID for one logical notification to one recipient.
// Otherwise Send generates a random ID. Reusing an ID for different notification
// data or anonymous recipients is unsupported. This option does not deduplicate
// external side effects. Empty or whitespace-only IDs are rejected.
func WithID(id string) SendOption {
	return sendOptionFunc(func(c *sendConfig) error {
		if strings.TrimSpace(id) == "" {
			return ErrInvalidID
		}
		c.id = id
		return nil
	})
}

func newSendConfig(options ...SendOption) (sendConfig, error) {
	var c sendConfig
	for _, option := range options {
		if option != nil {
			if err := option.applySend(&c); err != nil {
				return sendConfig{}, err
			}
		}
	}
	return c, nil
}
