package rabbitmq

import (
	"sync"

	"github.com/go-fries/fries/queue/v3"
	amqp "github.com/rabbitmq/amqp091-go"
)

type rabbitLease struct {
	task     *queue.Task
	delivery amqp.Delivery
	closer   interface {
		Close() error
	}

	closeOnce sync.Once
	closeErr  error
}

func (l *rabbitLease) Task() *queue.Task {
	if l == nil {
		return nil
	}
	return l.task
}

func (l *rabbitLease) close() error {
	if l == nil || l.closer == nil {
		return nil
	}
	l.closeOnce.Do(func() {
		l.closeErr = l.closer.Close()
	})
	return l.closeErr
}

func closeLease(lease queue.Lease) {
	l, ok := lease.(*rabbitLease)
	if !ok {
		return
	}
	_ = l.close()
}
