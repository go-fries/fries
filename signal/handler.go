package signal //nolint:revive

import (
	"os"
)

type Handler interface {
	Listen() []os.Signal
	Handle(os.Signal)
}
