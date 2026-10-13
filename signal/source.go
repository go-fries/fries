package signal

import (
	"os"
	"os/signal"
)

// signalSource is an internal testing seam, not a public backend abstraction.
// Sources must not close the supplied channel.
type signalSource interface {
	Notify(chan<- os.Signal, ...os.Signal)
	Stop(chan<- os.Signal)
}

type osSignalSource struct{}

func (osSignalSource) Notify(ch chan<- os.Signal, signals ...os.Signal) {
	signal.Notify(ch, signals...)
}

func (osSignalSource) Stop(ch chan<- os.Signal) {
	signal.Stop(ch)
}
