package multidriver

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"slices"
	"sync"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
)

var (
	ErrMissingWriter = errors.New("multi: missing writer driver")
	ErrClose         = errors.New("multi: close has errors")
)

type Driver struct {
	writer  dialect.Driver
	readers []dialect.Driver
	policy  Policy
	close   func() error
}

var _ dialect.Driver = (*Driver)(nil)

func New(opts ...Option) (*Driver, error) {
	d := &Driver{}

	for _, opt := range opts {
		opt(d)
	}

	if err := d.init(); err != nil {
		return nil, err
	}

	return d, nil
}

func (d *Driver) init() error {
	if d.writer == nil {
		return ErrMissingWriter
	}

	d.readers = slices.Clone(d.readers)
	// Capture explicitly configured resources before adding the reader fallback.
	drivers := append([]dialect.Driver{d.writer}, d.readers...)
	d.close = sync.OnceValue(func() error { return closeDrivers(drivers) })

	if len(d.readers) == 0 {
		d.readers = append(d.readers, d.writer)
	}

	if d.policy == nil {
		d.policy = RoundRobinPolicy()
	}

	return nil
}

func (d *Driver) Exec(ctx context.Context, query string, args, v any) error {
	return d.writer.Exec(ctx, query, args, v)
}

func (d *Driver) Query(ctx context.Context, query string, args, v any) error {
	if ent.QueryFromContext(ctx) == nil {
		return d.writer.Query(ctx, query, args, v)
	}

	return d.policy.Resolve(d.readers).Query(ctx, query, args, v)
}

func (d *Driver) Tx(ctx context.Context) (dialect.Tx, error) {
	return d.writer.Tx(ctx)
}

func (d *Driver) BeginTx(ctx context.Context, opts *sql.TxOptions) (dialect.Tx, error) {
	return d.writer.(interface {
		BeginTx(context.Context, *sql.TxOptions) (dialect.Tx, error)
	}).BeginTx(ctx, opts)
}

// Close closes the writer and readers in configuration order. Equal comparable
// driver values are closed once; non-comparable values are closed per explicit
// entry. Repeated and concurrent calls wait for and return the same result.
// Any errors are joined with ErrClose and can be inspected with errors.Is/As.
func (d *Driver) Close() error {
	return d.close()
}

func closeDrivers(drivers []dialect.Driver) error {
	seen := make(map[dialect.Driver]struct{}, len(drivers))
	var errs []error
	for _, driver := range drivers {
		// Value.Comparable also checks dynamic values inside interface fields.
		if reflect.ValueOf(driver).Comparable() {
			if _, ok := seen[driver]; ok {
				continue
			}
			seen[driver] = struct{}{}
		}
		if err := driver.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(append([]error{ErrClose}, errs...)...)
	}

	return nil
}

func (d *Driver) Dialect() string {
	return d.writer.Dialect()
}
