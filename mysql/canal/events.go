package canal

import (
	"context"

	"github.com/go-fries/fries/event/v3"
	"github.com/go-mysql-org/go-mysql/canal"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
)

// ====================================================
// Some events opened for Event Dispatcher
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

// ==========================================================================================
// EventHandler is a canal event handler that dispatches events to the event dispatcher.
// ==========================================================================================

type eventHandler struct {
	ctx        context.Context
	dispatcher *event.Dispatcher
}

var _ canal.EventHandler = (*eventHandler)(nil)

func newEventHandler(ctx context.Context, dispatcher *event.Dispatcher) *eventHandler {
	return &eventHandler{
		ctx:        ctx,
		dispatcher: dispatcher,
	}
}

func (eh *eventHandler) OnRotate(header *replication.EventHeader, rotateEvent *replication.RotateEvent) error {
	return eh.dispatcher.Dispatch(eh.ctx, &RotateEvent{
		Header:      header,
		RotateEvent: rotateEvent,
	})
}

func (eh *eventHandler) OnTableChanged(header *replication.EventHeader, schema string, table string) error {
	return eh.dispatcher.Dispatch(eh.ctx, &TableChangedEvent{
		Header: header,
		Schema: schema,
		Table:  table,
	})
}

func (eh *eventHandler) OnDDL(header *replication.EventHeader, nextPos mysql.Position, queryEvent *replication.QueryEvent) error {
	return eh.dispatcher.Dispatch(eh.ctx, &DDLEvent{
		Header:     header,
		NextPos:    nextPos,
		QueryEvent: queryEvent,
	})
}

func (eh *eventHandler) OnRow(rowsEvent *canal.RowsEvent) error {
	return eh.dispatcher.Dispatch(eh.ctx, &RowEvent{
		RowsEvent: rowsEvent,
	})
}

func (eh *eventHandler) OnXID(header *replication.EventHeader, nextPos mysql.Position) error {
	return eh.dispatcher.Dispatch(eh.ctx, &XIDEvent{
		Header:  header,
		NextPos: nextPos,
	})
}

func (eh *eventHandler) OnGTID(header *replication.EventHeader, gtidEvent mysql.BinlogGTIDEvent) error {
	return eh.dispatcher.Dispatch(eh.ctx, &GTIDEvent{
		Header:    header,
		GTIDEvent: gtidEvent,
	})
}

func (eh *eventHandler) OnPosSynced(header *replication.EventHeader, pos mysql.Position, set mysql.GTIDSet, force bool) error {
	return eh.dispatcher.Dispatch(eh.ctx, &PosSyncedEvent{
		Header: header,
		Pos:    pos,
		Set:    set,
		Force:  force,
	})
}

func (eh *eventHandler) OnRowsQueryEvent(e *replication.RowsQueryEvent) error {
	return eh.dispatcher.Dispatch(eh.ctx, &RowsQueryEvent{
		RowsQueryEvent: e,
	})
}

func (eh *eventHandler) String() string {
	return "eventHandler"
}
