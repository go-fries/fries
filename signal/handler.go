package signal

import (
	"context"
	"os"
)

// Handler processes subscribed operating system signals.
type Handler interface {
	// Listen returns the signals that should be routed to this handler.
	Listen() []os.Signal

	// Handle processes a received signal.
	Handle(context.Context, os.Signal)
}

type asyncable interface {
	Async() bool
}
