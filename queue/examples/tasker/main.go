package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-fries/fries/queue/adapter/memory/v3"
	"github.com/go-fries/fries/queue/v3"
)

const sendEmailTaskType = "send_email"

type SendEmail struct {
	UserID  int    `json:"user_id"`
	Subject string `json:"subject"`
}

type EmailSender interface {
	Send(ctx context.Context, userID int, subject string) error
}

type LogEmailSender struct{}

func (LogEmailSender) Send(_ context.Context, userID int, subject string) error {
	fmt.Printf("sent email: user_id=%d subject=%s\n", userID, subject)
	return nil
}

type SendEmailTasker struct {
	producer *queue.Producer
	sender   EmailSender
}

func NewSendEmailTasker(producer *queue.Producer, sender EmailSender) *SendEmailTasker {
	return &SendEmailTasker{
		producer: producer,
		sender:   sender,
	}
}

func (t *SendEmailTasker) TaskType() string {
	return sendEmailTaskType
}

func (t *SendEmailTasker) Enqueue(ctx context.Context, userID int, subject string, opts ...queue.EnqueueOption) (*queue.Task, error) {
	return queue.EnqueueFor(ctx, t.producer, t.TaskType(), SendEmail{
		UserID:  userID,
		Subject: subject,
	}, opts...)
}

func (t *SendEmailTasker) Handle(ctx context.Context, task *queue.TaskFor[SendEmail]) error {
	return t.sender.Send(ctx, task.Payload.UserID, task.Payload.Subject)
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	q := memory.NewQueue()
	tasker := NewSendEmailTasker(queue.NewProducer(q), LogEmailSender{})
	worker := queue.NewWorker(
		q,
		queue.HandleTasker[SendEmail](tasker),
		queue.WithPollInterval(10*time.Millisecond),
	)

	errs := make(chan error, 1)
	go func() {
		errs <- worker.Run(ctx)
	}()

	if _, err := tasker.Enqueue(ctx, 42, "welcome"); err != nil {
		log.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)

	cancel()
	if err := <-errs; err != nil {
		log.Fatal(err)
	}
}
