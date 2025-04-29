package internal

import (
	"context"
	"fmt"

	"github.com/go-mysql-org/go-mysql/canal"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
	"golang.org/x/sync/errgroup"
)

// ====================================================
// Some events opened for Dispatcher
// ====================================================

type RotateEvent struct {
	Header      *replication.EventHeader
	RotateEvent *replication.RotateEvent
}

type TableChangedEvent struct {
	Header *replication.EventHeader
	Schema string
	Table  string
}

type DDLEvent struct {
	Header     *replication.EventHeader
	NextPos    mysql.Position
	QueryEvent *replication.QueryEvent
}

type RowEvent struct {
	RowsEvent *canal.RowsEvent
}

type XIDEvent struct {
	Header  *replication.EventHeader
	NextPos mysql.Position
}

type GTIDEvent struct {
	Header    *replication.EventHeader
	GTIDEvent mysql.BinlogGTIDEvent
}

type PosSyncedEvent struct {
	Header *replication.EventHeader
	Pos    mysql.Position
	Set    mysql.GTIDSet
	Force  bool
}

type RowsQueryEvent struct {
	RowsQueryEvent *replication.RowsQueryEvent
}

// ====================================================
// Some listeners opened for Dispatcher
// ====================================================

type RotateListener interface {
	OnRotate(ctx context.Context, event *RotateEvent) error
}

type TableChangedListener interface {
	OnTableChanged(ctx context.Context, event *TableChangedEvent) error
}

type DDLListener interface {
	OnDDL(ctx context.Context, event *DDLEvent) error
}

type RowListener interface {
	OnRow(ctx context.Context, event *RowEvent) error
}

type XIDListener interface {
	OnXID(ctx context.Context, event *XIDEvent) error
}

type GTIDListener interface {
	OnGTID(ctx context.Context, event *GTIDEvent) error
}

type PosSyncedListener interface {
	OnPosSynced(ctx context.Context, event *PosSyncedEvent) error
}

type RowsQueryListener interface {
	OnRowsQuery(ctx context.Context, event *RowsQueryEvent) error
}

// ====================================================
// Dispatcher is a dispatcher for MySQL binlog events
// ====================================================

type Dispatcher struct {
	rotateListeners       []RotateListener
	tableChangedListeners []TableChangedListener
	ddlListeners          []DDLListener
	rowListeners          []RowListener
	xidListeners          []XIDListener
	gtidListeners         []GTIDListener
	posSyncedListeners    []PosSyncedListener
	rowsQueryListeners    []RowsQueryListener
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{}
}

func (d *Dispatcher) Registers(listeners ...any) {
	for _, listener := range listeners {
		if l, ok := listener.(RotateListener); ok {
			d.rotateListeners = append(d.rotateListeners, l)
		}

		if l, ok := listener.(TableChangedListener); ok {
			d.tableChangedListeners = append(d.tableChangedListeners, l)
		}

		if l, ok := listener.(DDLListener); ok {
			d.ddlListeners = append(d.ddlListeners, l)
		}

		if l, ok := listener.(RowListener); ok {
			d.rowListeners = append(d.rowListeners, l)
		}

		if l, ok := listener.(XIDListener); ok {
			d.xidListeners = append(d.xidListeners, l)
		}

		if l, ok := listener.(GTIDListener); ok {
			d.gtidListeners = append(d.gtidListeners, l)
		}

		if l, ok := listener.(PosSyncedListener); ok {
			d.posSyncedListeners = append(d.posSyncedListeners, l)
		}

		if l, ok := listener.(RowsQueryListener); ok {
			d.rowsQueryListeners = append(d.rowsQueryListeners, l)
		}
	}
}

func (d *Dispatcher) DispatchRotate(ctx context.Context, event *RotateEvent) error {
	return 	dispatch(ctx, d.rotateListeners, func(ctx context.Context, listener RotateListener) error {
		return listener.OnRotate(ctx, event)
	})
}

func (d *Dispatcher) DispatchTableChanged(ctx context.Context, event *TableChangedEvent) error {
	return dispatch(ctx, d.tableChangedListeners, func(ctx context.Context, listener TableChangedListener) error {
		return listener.OnTableChanged(ctx, event)
	})
}

func (d *Dispatcher) DispatchDDL(ctx context.Context, event *DDLEvent) error {
	return dispatch(ctx, d.ddlListeners, func(ctx context.Context, listener DDLListener) error {
		return listener.OnDDL(ctx, event)
	})
}

func (d *Dispatcher) DispatchRow(ctx context.Context, event *RowEvent) error {
	return dispatch(ctx, d.rowListeners, func(ctx context.Context, listener RowListener) error {
		return listener.OnRow(ctx, event)
	})
}

func (d *Dispatcher) DispatchXID(ctx context.Context, event *XIDEvent) error {
	return dispatch(ctx, d.xidListeners, func(ctx context.Context, listener XIDListener) error {
		return listener.OnXID(ctx, event)
	})
}

func (d *Dispatcher) DispatchGTID(ctx context.Context, event *GTIDEvent) error {
	return dispatch(ctx, d.gtidListeners, func(ctx context.Context, listener GTIDListener) error {
		return listener.OnGTID(ctx, event)
	})
}

func (d *Dispatcher) DispatchPosSynced(ctx context.Context, event *PosSyncedEvent) error {
	return dispatch(ctx, d.posSyncedListeners, func(ctx context.Context, listener PosSyncedListener) error {)
		return listener.OnPosSynced(ctx, event)
	})
}

func (d *Dispatcher) DispatchRowsQuery(ctx context.Context, event *RowsQueryEvent) error {=
	return dispatch(ctx, d.rowsQueryListeners, func(ctx context.Context, listener RowsQueryListener) error {
		return listener.OnRowsQuery(ctx, event)
	})
}

func dispatch[L any](ctx context.Context, listeners []L, callback func(ctx context.Context, listener L) error) error {
	eg, ctx := errgroup.WithContext(ctx)
	for _, listener := range listeners {
		eg.Go(func() error {
			return callback(ctx, listener)
		})
	}
	return eg.Wait()
}

// Deprecated: use [Dispatcher.DispatchRotate], [Dispatcher.DispatchTableChanged], etc. instead.
func (d *Dispatcher) Dispatch(ctx context.Context, event any) error {
	switch e := event.(type) {
	case *RotateEvent:
		return d.DispatchRotate(ctx, e)
	case *TableChangedEvent:
		return d.DispatchTableChanged(ctx, e)
	case *DDLEvent:
		return d.DispatchDDL(ctx, e)
	case *RowEvent:
		return d.DispatchRow(ctx, e)
	case *XIDEvent:
		return d.DispatchXID(ctx, e)
	case *GTIDEvent:
		return d.DispatchGTID(ctx, e)
	case *PosSyncedEvent:
		return d.DispatchPosSynced(ctx, e)
	case *RowsQueryEvent:
		return d.DispatchRowsQuery(ctx, e)
	default:
		return fmt.Errorf("unsupported event type: %T", event)
	}
}
