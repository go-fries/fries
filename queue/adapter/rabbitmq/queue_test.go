package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-fries/fries/queue/v3"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type declareCall struct {
	name       string
	durable    bool
	autoDelete bool
	exclusive  bool
	noWait     bool
	args       amqp.Table
}

type publishCall struct {
	exchange  string
	key       string
	mandatory bool
	immediate bool
	msg       amqp.Publishing
}

type fakeChannel struct {
	declareErr error
	publishErr error
	getErr     error
	deliveries []amqp.Delivery
	declares   []declareCall
	publishes  []publishCall
}

func (c *fakeChannel) QueueDeclare(
	name string,
	durable bool,
	autoDelete bool,
	exclusive bool,
	noWait bool,
	args amqp.Table,
) (amqp.Queue, error) {
	c.declares = append(c.declares, declareCall{
		name:       name,
		durable:    durable,
		autoDelete: autoDelete,
		exclusive:  exclusive,
		noWait:     noWait,
		args:       args,
	})
	if c.declareErr != nil {
		return amqp.Queue{}, c.declareErr
	}
	return amqp.Queue{Name: name}, nil
}

func (c *fakeChannel) PublishWithContext(
	_ context.Context,
	exchange string,
	key string,
	mandatory bool,
	immediate bool,
	msg amqp.Publishing,
) error {
	c.publishes = append(c.publishes, publishCall{
		exchange:  exchange,
		key:       key,
		mandatory: mandatory,
		immediate: immediate,
		msg:       msg,
	})
	return c.publishErr
}

func (c *fakeChannel) Get(string, bool) (amqp.Delivery, bool, error) {
	if c.getErr != nil {
		return amqp.Delivery{}, false, c.getErr
	}
	if len(c.deliveries) == 0 {
		return amqp.Delivery{}, false, nil
	}
	delivery := c.deliveries[0]
	c.deliveries = c.deliveries[1:]
	return delivery, true, nil
}

type fakeAcknowledger struct {
	acks []uint64
}

func (a *fakeAcknowledger) Ack(tag uint64, _ bool) error {
	a.acks = append(a.acks, tag)
	return nil
}

func (*fakeAcknowledger) Nack(uint64, bool, bool) error {
	return nil
}

func (*fakeAcknowledger) Reject(uint64, bool) error {
	return nil
}

func TestQueue_ConfigDefaultsAndOptions(t *testing.T) {
	t.Parallel()

	q := NewQueue(&fakeChannel{}, WithPrefix("app."), WithDurable(false), WithDelayQueueTTL(2*time.Hour))

	assert.Equal(t, "app.critical", q.readyQueueName("critical"))
	assert.Equal(t, "app.critical.dead", q.deadLetterQueueName("critical"))
	assert.Equal(t, "app.critical.delay.1500", q.delayQueueName("critical", 1500*time.Millisecond))
	assert.False(t, q.durable)
	assert.Equal(t, 2*time.Hour, q.delayQueueTTL)
}

func TestQueue_EnqueuePublishesReadyTask(t *testing.T) {
	t.Parallel()

	ch := &fakeChannel{}
	q := NewQueue(ch, WithPrefix("app"))
	task := &queue.Task{
		ID:      "task-1",
		Type:    "send_email",
		Queue:   "emails",
		Payload: []byte("hello"),
	}

	err := q.Enqueue(t.Context(), task)
	require.NoError(t, err)

	require.Len(t, ch.declares, 1)
	assert.Equal(t, "app.emails", ch.declares[0].name)
	assert.True(t, ch.declares[0].durable)

	require.Len(t, ch.publishes, 1)
	assert.Equal(t, "", ch.publishes[0].exchange)
	assert.Equal(t, "app.emails", ch.publishes[0].key)
	assert.Equal(t, amqp.Persistent, ch.publishes[0].msg.DeliveryMode)
	assert.Equal(t, contentTypeJSON, ch.publishes[0].msg.ContentType)
	assert.Equal(t, "task-1", ch.publishes[0].msg.MessageId)

	var stored queue.Task
	require.NoError(t, json.Unmarshal(ch.publishes[0].msg.Body, &stored))
	assert.Equal(t, "send_email", stored.Type)
	assert.Equal(t, []byte("hello"), stored.Payload)
}

func TestQueue_EnqueuePublishesDelayedTask(t *testing.T) {
	t.Parallel()

	ch := &fakeChannel{}
	q := NewQueue(ch, WithDelayQueueTTL(2*time.Second))
	task := &queue.Task{
		ID:    "task-1",
		Type:  "send_email",
		Queue: "emails",
	}

	err := q.publishTask(t.Context(), task, 1500*time.Millisecond)
	require.NoError(t, err)

	require.Len(t, ch.declares, 2)
	assert.Equal(t, "emails", ch.declares[0].name)
	assert.Equal(t, "emails.delay.1500", ch.declares[1].name)
	assert.Equal(t, int64(1500), ch.declares[1].args["x-message-ttl"])
	assert.Equal(t, int64(3500), ch.declares[1].args["x-expires"])
	assert.Equal(t, "", ch.declares[1].args["x-dead-letter-exchange"])
	assert.Equal(t, "emails", ch.declares[1].args["x-dead-letter-routing-key"])

	require.Len(t, ch.publishes, 1)
	assert.Equal(t, "emails.delay.1500", ch.publishes[0].key)
}

func TestQueue_DequeueReturnsNoTask(t *testing.T) {
	t.Parallel()

	ch := &fakeChannel{}
	q := NewQueue(ch)

	lease, err := q.Dequeue(t.Context(), "", time.Minute)

	require.ErrorIs(t, err, queue.ErrNoTask)
	assert.Nil(t, lease)
	require.Len(t, ch.declares, 1)
	assert.Equal(t, "default", ch.declares[0].name)
}

func TestQueue_DequeueDecodesTaskAndAck(t *testing.T) {
	t.Parallel()

	task := &queue.Task{ID: "task-1", Type: "send_email", Queue: "emails", Attempt: 2}
	body, err := json.Marshal(task)
	require.NoError(t, err)
	ack := &fakeAcknowledger{}
	ch := &fakeChannel{
		deliveries: []amqp.Delivery{{
			Body:         body,
			DeliveryTag:  42,
			Acknowledger: ack,
		}},
	}
	q := NewQueue(ch, WithPrefix("app"))

	lease, err := q.Dequeue(t.Context(), "emails", time.Minute)
	require.NoError(t, err)
	require.NotNil(t, lease)
	assert.Equal(t, 3, lease.Task().Attempt)

	require.NoError(t, q.Ack(t.Context(), lease))
	assert.Equal(t, []uint64{42}, ack.acks)
}

func TestQueue_RetryPublishesDelayedTaskAndAcksOriginal(t *testing.T) {
	t.Parallel()

	ack := &fakeAcknowledger{}
	ch := &fakeChannel{}
	q := NewQueue(ch)
	lease := &rabbitLease{
		task: &queue.Task{
			ID:    "task-1",
			Type:  "send_email",
			Queue: "emails",
		},
		delivery: amqp.Delivery{DeliveryTag: 7, Acknowledger: ack},
	}

	err := q.Retry(t.Context(), lease, 2*time.Second)
	require.NoError(t, err)

	require.Len(t, ch.declares, 2)
	assert.Equal(t, "emails", ch.declares[0].name)
	assert.Equal(t, "emails.delay.2000", ch.declares[1].name)
	assert.Equal(t, int64(2000), ch.declares[1].args["x-message-ttl"])
	require.Len(t, ch.publishes, 1)
	assert.Equal(t, "emails.delay.2000", ch.publishes[0].key)
	assert.Equal(t, []uint64{7}, ack.acks)
}

func TestQueue_DeadLetterPublishesReasonAndAcksOriginal(t *testing.T) {
	t.Parallel()

	ack := &fakeAcknowledger{}
	ch := &fakeChannel{}
	q := NewQueue(ch)
	lease := &rabbitLease{
		task: &queue.Task{
			ID:    "task-1",
			Type:  "send_email",
			Queue: "emails",
		},
		delivery: amqp.Delivery{DeliveryTag: 8, Acknowledger: ack},
	}

	err := q.DeadLetter(t.Context(), lease, "retry exhausted")
	require.NoError(t, err)

	require.Len(t, ch.declares, 1)
	assert.Equal(t, "emails.dead", ch.declares[0].name)
	require.Len(t, ch.publishes, 1)
	assert.Equal(t, "emails.dead", ch.publishes[0].key)
	assert.Equal(t, "retry exhausted", ch.publishes[0].msg.Headers[deadReasonKey])
	assert.Equal(t, []uint64{8}, ack.acks)

	var stored queue.Task
	require.NoError(t, json.Unmarshal(ch.publishes[0].msg.Body, &stored))
	assert.Equal(t, "retry exhausted", stored.Metadata[deadReasonKey])
}

func TestQueue_NilChannelReturnsError(t *testing.T) {
	t.Parallel()

	q := NewQueue(nil)

	err := q.Enqueue(t.Context(), &queue.Task{Type: "send_email"})

	require.ErrorIs(t, err, errNilChannel)
}

func TestQueue_PropagatesChannelErrors(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("declare failed")
	q := NewQueue(&fakeChannel{declareErr: wantErr})

	err := q.Enqueue(t.Context(), &queue.Task{Type: "send_email"})

	require.ErrorIs(t, err, wantErr)
}
