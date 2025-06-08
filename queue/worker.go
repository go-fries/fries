package queue

import (
	"context"

	"github.com/go-fries/fries/codec/v3"
)

type JobHandler interface {
	// Handle processes the job with the given data.
	Handle(ctx context.Context, data []byte) error
}

type Worker struct {
	queue        string
	queuer       Queuer
	codec        codec.Codec
	handlers     map[string]JobHandler
	workerNumber int
}

func NewWorker(queuer Queuer) *Worker {
	return &Worker{
		queue:    "default",
		queuer:   queuer,
		codec:    nil,
		handlers: make(map[string]JobHandler),
	}
}

func (w *Worker) RegisterHandler(subject string, handler JobHandler) {
	w.handlers[subject] = handler
}

func (w *Worker) Start(ctx context.Context) error {
	for i := 0; i < w.workerNumber; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					data, err := w.queuer.Dequeue(ctx, w.queue)
					if err != nil {
						// Handle error (e.g., log it)
						continue
					}
					if len(data) == 0 {
						continue // No data to process
					}

					// Decode the job data
					var message Message
					if err := w.codec.Unmarshal(data, &message); err != nil {
						// Handle decoding error (e.g., log it)
						continue
					}

					job, err := parseMessage(message)
					if err != nil {
						_ = w.queuer.Enqueue(ctx, w.queue, data) // Optionally re-queue the job
						// todo: log error
					}

					handler, exists := w.handlers[job.Subject()]
					if !exists {
						_ = w.queuer.Enqueue(ctx, w.queue, data) // Optionally re-queue the job
						// todo: log error
					}

					if err := handler.Handle(ctx, job.Data()); err != nil {
						// job.RetryCount++ //todo: handle retry logic
						_ = w.queuer.Enqueue(ctx, w.queue, data) // Optionally re-queue the job
					}
				}
			}
		}()
	}

	return nil
}

func (w *Worker) Stop(ctx context.Context) error {
	return nil
}
