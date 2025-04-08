package canal

import (
	"context"
	"fmt"

	"github.com/go-fries/fries/mysql/canal/v3/internal"
	orgcanal "github.com/go-mysql-org/go-mysql/canal"
	"github.com/go-mysql-org/go-mysql/mysql"
)

type Config struct {
	Addr     string // host
	User     string // username
	Password string // password
	Charset  string
	Flavor   string
}

type Canal struct {
	config *Config
	canal  *orgcanal.Canal

	// event dispatcher
	dispatcher *internal.Dispatcher
}

type Option func(*Canal) error

// WithListeners registers listeners to the canal dispatcher.
// The listeners must implement the corresponding listener interfaces.
func WithListeners(listeners ...any) Option {
	return func(c *Canal) error {
		if len(listeners) > 0 {
			c.dispatcher.Registers(listeners...)
		}
		return nil
	}
}

func New(config *Config, opts ...Option) (*Canal, error) {
	c := &Canal{
		config:     config,
		dispatcher: internal.NewDispatcher(),
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (c *Canal) Start(ctx context.Context) error {
	if err := c.initCanal(ctx); err != nil {
		return err
	}

	position, err := c.canal.GetMasterPos()
	if err != nil {
		return err
	}

	return c.canal.RunFrom(position)
}

func (c *Canal) Stop(ctx context.Context) error {
	done := make(chan struct{})

	go func() {
		defer close(done)
		c.canal.Close()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		// Canal closed successfully
		return nil
	}
}

func (c *Canal) initCanal(ctx context.Context) error {
	if c.config.Addr == "" {
		return fmt.Errorf("canal: addr is empty")
	}

	if c.config.User == "" {
		return fmt.Errorf("canal: username is empty")
	}

	if c.config.Password == "" {
		return fmt.Errorf("canal: password is empty")
	}

	cfg := orgcanal.NewDefaultConfig()
	cfg.Addr = c.config.Addr
	cfg.User = c.config.User
	cfg.Password = c.config.Password

	if c.config.Charset != "" {
		cfg.Charset = c.config.Charset
	}

	if c.config.Flavor != "" {
		cfg.Flavor = c.config.Flavor
	} else {
		cfg.Flavor = mysql.MySQLFlavor // 默认使用mysql
	}

	cnl, err := orgcanal.NewCanal(cfg)
	if err != nil {
		return fmt.Errorf("canal: new canal error: %w", err)
	}

	// set event handler
	cnl.SetEventHandler(newEventHandler(ctx, c.dispatcher))

	c.canal = cnl
	return nil
}
