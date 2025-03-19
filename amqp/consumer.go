package amqp

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

type ConsumerMessage interface {
	Consume(ctx context.Context, delivery amqp.Delivery) error
	GetQueue() string
	GetConcurrent() int
}

type Consumer struct {
	conn *amqp.Connection
}

func NewConsumer(conn *amqp.Connection) *Consumer {
	return &Consumer{
		conn: conn,
	}
}

func (c *Consumer) Consume(ctx context.Context, msg ConsumerMessage) error {
	channel, err := c.conn.Channel()
	if err != nil {
		return err
	}

	_, err = channel.QueueDeclare(
		msg.GetQueue(),
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	deliveries, err := channel.Consume(
		msg.GetQueue(),
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	ch := make(chan struct{}, msg.GetConcurrent())

	for delivery := range deliveries {
		ch <- struct{}{}
		go func() {
			defer func() {
				<-ch
			}()

			if err := msg.Consume(ctx, delivery); err != nil {
				log.Errorf("consume message error: %v", err)
			}
		}()
	}

	return nil
}
