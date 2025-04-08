package canal

import (
	"context"
	"errors"

	"github.com/go-mysql-org/go-mysql/mysql"
)

var ErrPositionNotFound = errors.New("canal: position not found")

type Positioner interface {
	Get(ctx context.Context) (mysql.Position, error)
	Set(ctx context.Context, pos mysql.Position) error
}

// positionListener is a listener that listens for position sync events and updates the positioner.
// This listener will be automatically injected into Canal by default to update relevant information.
type positionListener struct {
	positioner Positioner
}

var _ PosSyncedListener = (*positionListener)(nil)

func newPositionListener(positioner Positioner) *positionListener {
	return &positionListener{
		positioner: positioner,
	}
}

func (p *positionListener) OnPosSynced(ctx context.Context, event *PosSyncedEvent) error {
	return p.positioner.Set(ctx, event.Pos)
}
