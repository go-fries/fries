package rabbitmq

import (
	"github.com/go-fries/fries/queue/v3"
	amqp "github.com/rabbitmq/amqp091-go"
)

type rabbitLease struct {
	task     *queue.Task
	delivery amqp.Delivery
}

func (l *rabbitLease) Task() *queue.Task {
	if l == nil {
		return nil
	}
	return l.task
}
