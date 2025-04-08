package canal

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/go-fries/fries/mysql/canal/v3/internal"
	orgcanal "github.com/go-mysql-org/go-mysql/canal"
	"github.com/go-mysql-org/go-mysql/mysql"
)

var logger = slog.New(slog.NewTextHandler(os.Stderr, nil))

type Config struct {
	Addr     string // host
	User     string // username
	Password string // password
	Charset  string // charset
	Flavor   string // support mysql, mariadb

	// include/exclude tables
	// 	IncludeTablesRegex : [".*\\.canal"], ExcludeTablesRegex : ["mysql\\..*"]
	IncludeTablesRegex []string
	ExcludeTablesRegex []string
}

type Canal struct {
	config *Config
	canal  *orgcanal.Canal

	positioner Positioner

	// event dispatcher
	dispatcher *internal.Dispatcher
}

type Option func(*Canal) error

// WithPositioner sets the positioner for the canal.
func WithPositioner(positioner Positioner) Option {
	return func(c *Canal) error {
		if positioner == nil {
			return fmt.Errorf("canal: positioner is nil")
		}
		c.positioner = positioner
		return nil
	}
}

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

	c.initPositioner()

	return c, nil
}

func (c *Canal) Start(ctx context.Context) error {
	if err := c.initCanal(ctx); err != nil {
		return err
	}

	position, err := c.getStartPosition(ctx)
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

// GetDelay returns the delay of the canal in duration.
func (c *Canal) GetDelay() time.Duration {
	return time.Duration(c.canal.GetDelay()) * time.Second
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

	if len(c.config.IncludeTablesRegex) > 0 {
		cfg.IncludeTableRegex = c.config.IncludeTablesRegex
	}

	if len(c.config.ExcludeTablesRegex) > 0 {
		cfg.ExcludeTableRegex = c.config.ExcludeTablesRegex
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

func (c *Canal) initPositioner() {
	if c.positioner == nil {
		return
	}

	c.dispatcher.Registers(newPositionListener(c.positioner))
}

// getStartPosition returns the start position for the canal.
// If extracted from Positioner, use it; otherwise, default to using the latest Position.
func (c *Canal) getStartPosition(ctx context.Context) (pos mysql.Position, err error) {
	if c.positioner != nil {
		pos, err = c.positioner.Get(ctx)
		if err == nil && pos.Name != "" && pos.Pos != 0 {
			return pos, nil
		} else {
			logger.Warn("canal: positioner get error, using the latest position")
		}
	}

	return c.canal.GetMasterPos()
}
