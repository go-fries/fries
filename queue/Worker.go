package queue

import (
	"context"
	"sync"
)

type Worker struct {
	queue      Queue
	concurrent int
}

func NewWorker(queue Queue) *Worker {
	return &Worker{
		queue:      queue,
		concurrent: 10,
	}
}

func (w *Worker) Start(ctx context.Context) error {
	wg := sync.WaitGroup{}
	for i := 0; i < w.concurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					task, err := w.queue.Dequeue(ctx, "task")
					if err != nil {
						continue
					}
					// Process the task
					_ = task // Replace with actual processing logic
				}
			}
		}()
	}
	wg.Wait()
	return nil
}

func (w *Worker) Stop(ctx context.Context) error {
	return nil
}
