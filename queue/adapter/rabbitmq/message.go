package rabbitmq

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/go-fries/fries/queue/v3"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	contentTypeJSON = "application/json"
	deadReasonKey   = "queue.dead_letter.reason"
)

func publishingFromTask(task *queue.Task) (amqp.Publishing, error) {
	data, err := json.Marshal(task)
	if err != nil {
		return amqp.Publishing{}, err
	}

	return amqp.Publishing{
		ContentType:  contentTypeJSON,
		DeliveryMode: amqp.Persistent,
		MessageId:    task.ID,
		Timestamp:    task.CreatedAt,
		Body:         data,
	}, nil
}

func deliveryFromAMQP(amqpDelivery amqp.Delivery) (delivery, error) {
	var task queue.Task
	if err := json.Unmarshal(amqpDelivery.Body, &task); err != nil {
		return delivery{}, err
	}
	if task.Queue == "" {
		task.Queue = queue.DefaultQueue
	}
	task.Attempt = nextAttempt(task.Attempt)
	return delivery{
		task:     &task,
		delivery: amqpDelivery,
	}, nil
}

func nextAttempt(attempt int) int {
	if attempt < 0 {
		return 1
	}
	if attempt == math.MaxInt {
		return math.MaxInt
	}
	return attempt + 1
}

func delayFromAvailableAt(now, availableAt time.Time) time.Duration {
	if availableAt.IsZero() || !availableAt.After(now) {
		return 0
	}
	return availableAt.Sub(now)
}

func durationMillis(d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	if d < time.Millisecond {
		return 1
	}
	return d.Milliseconds()
}

func queueDeclareError(name string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("queue/adapter/rabbitmq: declare queue %q: %w", name, err)
}
