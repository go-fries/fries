package canal

import (
	"context"

	"github.com/go-fries/fries/mysql/canal/v3/internal"
	"github.com/go-mysql-org/go-mysql/canal"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
)

// ====================================================
// Some events opened for Dispatcher
// ====================================================

type (
	RotateEvent       = internal.RotateEvent
	TableChangedEvent = internal.TableChangedEvent
	DDLEvent          = internal.DDLEvent
	RowEvent          = internal.RowEvent
	XIDEvent          = internal.XIDEvent
	GTIDEvent         = internal.GTIDEvent
	PosSyncedEvent    = internal.PosSyncedEvent
	RowsQueryEvent    = internal.RowsQueryEvent
)

// ====================================================
// Some listeners opened for Dispatcher
// ====================================================

type (
	RotateListener       = internal.RotateListener
	TableChangedListener = internal.TableChangedListener
	DDLListener          = internal.DDLListener
	RowListener          = internal.RowListener
	XIDListener          = internal.XIDListener
	GTIDListener         = internal.GTIDListener
	PosSyncedListener    = internal.PosSyncedListener
	RowsQueryListener    = internal.RowsQueryListener
)

// ==========================================================================================
// EventHandler is a canal event handler that dispatches events to the dispatcher.
// ==========================================================================================

type eventHandler struct {
	ctx        context.Context
	dispatcher *internal.Dispatcher
}

var _ canal.EventHandler = (*eventHandler)(nil)

func newEventHandler(ctx context.Context, dispatcher *internal.Dispatcher) *eventHandler {
	return &eventHandler{
		ctx:        ctx,
		dispatcher: dispatcher,
	}
}

func (eh *eventHandler) OnRotate(header *replication.EventHeader, rotateEvent *replication.RotateEvent) error {
	return eh.dispatcher.DispatchRotate(eh.ctx, &RotateEvent{
		Header:      header,
		RotateEvent: rotateEvent,
	})
}

func (eh *eventHandler) OnTableChanged(header *replication.EventHeader, schema string, table string) error {
	return eh.dispatcher.DispatchTableChanged(eh.ctx, &TableChangedEvent{
		Header: header,
		Schema: schema,
		Table:  table,
	})
}

func (eh *eventHandler) OnDDL(header *replication.EventHeader, nextPos mysql.Position, queryEvent *replication.QueryEvent) error {
	return eh.dispatcher.DispatchDDL(eh.ctx, &DDLEvent{
		Header:     header,
		NextPos:    nextPos,
		QueryEvent: queryEvent,
	})
}

func (eh *eventHandler) OnRow(rowsEvent *canal.RowsEvent) error {
	return eh.dispatcher.DispatchRow(eh.ctx, &RowEvent{
		RowsEvent: rowsEvent,
	})
}

func (eh *eventHandler) OnXID(header *replication.EventHeader, nextPos mysql.Position) error {
	return eh.dispatcher.DispatchXID(eh.ctx, &XIDEvent{
		Header:  header,
		NextPos: nextPos,
	})
}

func (eh *eventHandler) OnGTID(header *replication.EventHeader, gtidEvent mysql.BinlogGTIDEvent) error {
	return eh.dispatcher.DispatchGTID(eh.ctx, &GTIDEvent{
		Header:    header,
		GTIDEvent: gtidEvent,
	})
}

func (eh *eventHandler) OnPosSynced(header *replication.EventHeader, pos mysql.Position, set mysql.GTIDSet, force bool) error {
	return eh.dispatcher.DispatchPosSynced(eh.ctx, &PosSyncedEvent{
		Header: header,
		Pos:    pos,
		Set:    set,
		Force:  force,
	})
}

func (eh *eventHandler) OnRowsQueryEvent(e *replication.RowsQueryEvent) error {
	return eh.dispatcher.DispatchRowsQuery(eh.ctx, &RowsQueryEvent{
		RowsQueryEvent: e,
	})
}

func (eh *eventHandler) String() string {
	return "canal:handler"
}
