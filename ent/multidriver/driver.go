package multidriver

import (
	"context"
	"database/sql"
	"errors"

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

func (d *Driver) Close() error {
	var errs []error
	if err := d.writer.Close(); err != nil {
		errs = append(errs, err)
	}

	for _, r := range d.readers {
		if err := r.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return ErrClose
	}

	return nil
}

func (d *Driver) Dialect() string {
	return d.writer.Dialect()
}
