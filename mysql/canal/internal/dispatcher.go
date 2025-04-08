package internal

import (
	"context"
	"fmt"

	"github.com/go-mysql-org/go-mysql/canal"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
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
	for _, listener := range d.rotateListeners {
		if err := listener.OnRotate(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dispatcher) DispatchTableChanged(ctx context.Context, event *TableChangedEvent) error {
	for _, listener := range d.tableChangedListeners {
		if err := listener.OnTableChanged(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dispatcher) DispatchDDL(ctx context.Context, event *DDLEvent) error {
	for _, listener := range d.ddlListeners {
		if err := listener.OnDDL(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dispatcher) DispatchRow(ctx context.Context, event *RowEvent) error {
	for _, listener := range d.rowListeners {
		if err := listener.OnRow(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dispatcher) DispatchXID(ctx context.Context, event *XIDEvent) error {
	for _, listener := range d.xidListeners {
		if err := listener.OnXID(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dispatcher) DispatchGTID(ctx context.Context, event *GTIDEvent) error {
	for _, listener := range d.gtidListeners {
		if err := listener.OnGTID(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dispatcher) DispatchPosSynced(ctx context.Context, event *PosSyncedEvent) error {
	for _, listener := range d.posSyncedListeners {
		if err := listener.OnPosSynced(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dispatcher) DispatchRowsQuery(ctx context.Context, event *RowsQueryEvent) error {
	for _, listener := range d.rowsQueryListeners {
		if err := listener.OnRowsQuery(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

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
