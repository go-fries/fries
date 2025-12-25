package channel

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-fries/fries/queue/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDriver_PushAndPop(t *testing.T) {
	d := New()
	ctx := context.Background()

	require.NoError(t, d.Start(ctx))
	defer d.Stop(ctx)

	// Push a job
	job := queue.NewJob("test payload", queue.WithQueue("test"))
	require.NoError(t, d.Push(ctx, job))

	// Pop the job
	ctx2, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	popped, err := d.Pop(ctx2, "test")
	require.NoError(t, err)
	assert.Equal(t, job.ID(), popped.ID())
	assert.Equal(t, "test payload", popped.Payload())
}

func TestDriver_Priority(t *testing.T) {
	d := New()
	ctx := context.Background()

	require.NoError(t, d.Start(ctx))
	defer d.Stop(ctx)

	// Push jobs with different priorities
	lowPriority := queue.NewJob("low", queue.WithQueue("test"), queue.WithPriority(1))
	highPriority := queue.NewJob("high", queue.WithQueue("test"), queue.WithPriority(10))
	mediumPriority := queue.NewJob("medium", queue.WithQueue("test"), queue.WithPriority(5))

	require.NoError(t, d.Push(ctx, lowPriority))
	require.NoError(t, d.Push(ctx, highPriority))
	require.NoError(t, d.Push(ctx, mediumPriority))

	// Pop should return in priority order
	ctx2, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	job1, err := d.Pop(ctx2, "test")
	require.NoError(t, err)
	assert.Equal(t, "high", job1.Payload())

	job2, err := d.Pop(ctx2, "test")
	require.NoError(t, err)
	assert.Equal(t, "medium", job2.Payload())

	job3, err := d.Pop(ctx2, "test")
	require.NoError(t, err)
	assert.Equal(t, "low", job3.Payload())
}

func TestDriver_DelayedJob(t *testing.T) {
	d := New()
	ctx := context.Background()

	require.NoError(t, d.Start(ctx))
	defer d.Stop(ctx)

	// Push a delayed job
	job := queue.NewJob("delayed", queue.WithQueue("test"), queue.WithDelay(200*time.Millisecond))
	require.NoError(t, d.Push(ctx, job))

	// Immediate pop should timeout
	ctx2, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	popped, _ := d.Pop(ctx2, "test")
	cancel()
	assert.Nil(t, popped, "Expected no job to be available immediately")

	// Wait for job to become available
	time.Sleep(150 * time.Millisecond)

	ctx3, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	popped, err := d.Pop(ctx3, "test")
	require.NoError(t, err)
	assert.Equal(t, "delayed", popped.Payload())
}

func TestDriver_FailAndRetry(t *testing.T) {
	d := New()
	ctx := context.Background()

	require.NoError(t, d.Start(ctx))
	defer d.Stop(ctx)

	// Push a job with max 2 attempts
	job := queue.NewJob("retry-test", queue.WithQueue("test"), queue.WithMaxAttempts(2))
	require.NoError(t, d.Push(ctx, job))

	// Pop and fail it
	ctx2, cancel := context.WithTimeout(ctx, time.Second)
	popped, err := d.Pop(ctx2, "test")
	cancel()
	require.NoError(t, err)

	require.NoError(t, d.Fail(ctx, popped, nil))

	// Job should be back in queue (delayed)
	time.Sleep(1100 * time.Millisecond) // Wait for exponential backoff (1^2 = 1 second)

	ctx3, cancel := context.WithTimeout(ctx, time.Second)
	popped, err = d.Pop(ctx3, "test")
	cancel()
	require.NoError(t, err)
	assert.Equal(t, 1, popped.Attempts())

	// Fail again, should go to dead queue
	require.NoError(t, d.Fail(ctx, popped, nil))

	dead, err := d.Dead(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, int64(1), dead)
}

func TestDriver_DeadLetterQueue(t *testing.T) {
	d := New()
	ctx := context.Background()

	require.NoError(t, d.Start(ctx))
	defer d.Stop(ctx)

	// Push and fail a job with maxAttempts=1
	job := queue.NewJob("dead-test", queue.WithQueue("test"), queue.WithMaxAttempts(1))
	require.NoError(t, d.Push(ctx, job))

	ctx2, cancel := context.WithTimeout(ctx, time.Second)
	popped, err := d.Pop(ctx2, "test")
	cancel()
	require.NoError(t, err)

	require.NoError(t, d.Fail(ctx, popped, nil))

	// Check dead queue
	dead, err := d.Dead(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, int64(1), dead)

	// Peek dead queue
	deadJobs, err := d.PeekDead(ctx, "test", 10)
	require.NoError(t, err)
	assert.Len(t, deadJobs, 1)

	// Retry the dead job
	require.NoError(t, d.Retry(ctx, "test", popped.ID()))

	dead, err = d.Dead(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, int64(0), dead)

	size, err := d.Size(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, int64(1), size)
}

func TestDriver_Size(t *testing.T) {
	d := New()
	ctx := context.Background()

	require.NoError(t, d.Start(ctx))
	defer d.Stop(ctx)

	// Initially empty
	size, err := d.Size(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, int64(0), size)

	// Push jobs
	for i := 0; i < 5; i++ {
		require.NoError(t, d.Push(ctx, queue.NewJob(i, queue.WithQueue("test"))))
	}

	size, err = d.Size(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, int64(5), size)

	// Pop one
	ctx2, cancel := context.WithTimeout(ctx, time.Second)
	_, err = d.Pop(ctx2, "test")
	cancel()
	require.NoError(t, err)

	size, err = d.Size(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, int64(4), size)
}

func TestDriver_Clear(t *testing.T) {
	d := New()
	ctx := context.Background()

	require.NoError(t, d.Start(ctx))
	defer d.Stop(ctx)

	// Push jobs
	for i := 0; i < 3; i++ {
		require.NoError(t, d.Push(ctx, queue.NewJob(i, queue.WithQueue("test"))))
	}

	// Clear
	require.NoError(t, d.Clear(ctx, "test"))

	size, err := d.Size(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, int64(0), size)
}

func TestDriver_ConcurrentPushPop(t *testing.T) {
	d := New()
	ctx := context.Background()

	require.NoError(t, d.Start(ctx))
	defer d.Stop(ctx)

	const numJobs = 100
	var wg sync.WaitGroup
	var processed int64

	// Start consumers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				ctx2, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
				job, err := d.Pop(ctx2, "test")
				cancel()

				if err != nil || job == nil {
					if atomic.LoadInt64(&processed) >= numJobs {
						return
					}
					continue
				}

				atomic.AddInt64(&processed, 1)
			}
		}()
	}

	// Push jobs
	for i := 0; i < numJobs; i++ {
		require.NoError(t, d.Push(ctx, queue.NewJob(i, queue.WithQueue("test"))))
	}

	// Wait for all jobs to be processed
	time.Sleep(500 * time.Millisecond)

	// Stop consumers
	wg.Wait()

	assert.Equal(t, int64(numJobs), processed)
}

func TestDriver_MultipleQueues(t *testing.T) {
	d := New()
	ctx := context.Background()

	require.NoError(t, d.Start(ctx))
	defer d.Stop(ctx)

	// Push to different queues
	require.NoError(t, d.Push(ctx, queue.NewJob("queue1-job", queue.WithQueue("queue1"))))
	require.NoError(t, d.Push(ctx, queue.NewJob("queue2-job", queue.WithQueue("queue2"))))

	// Pop from queue1
	ctx2, cancel := context.WithTimeout(ctx, time.Second)
	job, err := d.Pop(ctx2, "queue1")
	cancel()
	require.NoError(t, err)
	assert.Equal(t, "queue1-job", job.Payload())

	// Pop from queue2
	ctx3, cancel := context.WithTimeout(ctx, time.Second)
	job, err = d.Pop(ctx3, "queue2")
	cancel()
	require.NoError(t, err)
	assert.Equal(t, "queue2-job", job.Payload())
}

func TestDriver_PopFromMultipleQueues(t *testing.T) {
	d := New()
	ctx := context.Background()

	require.NoError(t, d.Start(ctx))
	defer d.Stop(ctx)

	// Push to different queues with different priorities
	require.NoError(t, d.Push(ctx, queue.NewJob("q1-low", queue.WithQueue("queue1"), queue.WithPriority(1))))
	require.NoError(t, d.Push(ctx, queue.NewJob("q2-high", queue.WithQueue("queue2"), queue.WithPriority(10))))

	// Pop from both queues - should get highest priority across all queues
	ctx2, cancel := context.WithTimeout(ctx, time.Second)
	job, err := d.Pop(ctx2, "queue1", "queue2")
	cancel()
	require.NoError(t, err)
	assert.Equal(t, "q2-high", job.Payload())
}

func TestDriver_RetryAll(t *testing.T) {
	d := New()
	ctx := context.Background()

	require.NoError(t, d.Start(ctx))
	defer d.Stop(ctx)

	// Push and fail multiple jobs
	for i := 0; i < 3; i++ {
		job := queue.NewJob(i, queue.WithQueue("test"), queue.WithMaxAttempts(1))
		require.NoError(t, d.Push(ctx, job))

		ctx2, cancel := context.WithTimeout(ctx, time.Second)
		popped, err := d.Pop(ctx2, "test")
		cancel()
		require.NoError(t, err)

		require.NoError(t, d.Fail(ctx, popped, nil))
	}

	// Check dead queue
	dead, err := d.Dead(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, int64(3), dead)

	// Retry all
	count, err := d.RetryAll(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)

	// Verify dead queue is empty
	dead, err = d.Dead(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, int64(0), dead)

	// Verify jobs are back in queue
	size, err := d.Size(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, int64(3), size)
}
