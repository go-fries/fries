package eventbus

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var ctx = context.Background()

func TestEventBus_Func(t *testing.T) {
	topic := NewEvent[int]()
	ch := make(chan int, 1)

	assert.Equal(t, topic.ListenersCount(), 0)
	listener := topic.On(HandlerFunc[int](func(_ context.Context, msg int) error {
		ch <- msg
		return nil
	}))
	assert.Len(t, topic.Listeners(), 1)
	assert.Equal(t, 1, topic.ListenersCount())
	assert.NoError(t, topic.Emit(ctx, 1))
	assert.Equal(t, 1, <-ch)

	assert.NoError(t, listener.Off())
	assert.Len(t, topic.Listeners(), 0)
	assert.Equal(t, 0, topic.ListenersCount())
	assert.NoError(t, topic.Emit(ctx, 2))
	assert.Len(t, ch, 0)

	topic.OffAll()
	assert.NoError(t, topic.Emit(ctx, 3))
	assert.Len(t, ch, 0)
}

type MyEvent struct {
	ID int
}

func TestEventBus_Struct(t *testing.T) {
	topic := NewEvent[MyEvent]()
	ch := make(chan int, 1)

	listener := topic.On(HandlerFunc[MyEvent](func(_ context.Context, msg MyEvent) error {
		ch <- msg.ID
		return nil
	}))
	assert.NoError(t, topic.Emit(ctx, MyEvent{ID: 1}))
	assert.Equal(t, 1, <-ch)

	assert.NoError(t, listener.Off())
	assert.NoError(t, topic.Emit(ctx, MyEvent{ID: 2}))
	assert.Len(t, ch, 0)

	assert.ErrorIs(t, listener.Off(), ErrListenerNotFound)
}

func TestEventBus_ListenersReturnsCopy(t *testing.T) {
	topic := NewEvent[int]()

	listeners := topic.Listeners()
	assert.NotNil(t, listeners)
	assert.Empty(t, listeners)

	first := topic.On(HandlerFunc[int](func(context.Context, int) error {
		return nil
	}))
	second := topic.On(HandlerFunc[int](func(context.Context, int) error {
		return nil
	}))

	listeners = topic.Listeners()
	assert.Equal(t, []*Listener[int]{first, second}, listeners)

	listeners[0] = nil
	assert.Equal(t, []*Listener[int]{first, second}, topic.Listeners())

	topic.OffAll()
	listeners = topic.Listeners()
	assert.NotNil(t, listeners)
	assert.Empty(t, listeners)
}

func TestEventBus_SkipErrors(t *testing.T) {
	topic := NewEvent[int]()
	ch := make(chan int, 1)

	_ = topic.On(HandlerFunc[int](func(context.Context, int) error {
		return assert.AnError
	}))
	_ = topic.On(HandlerFunc[int](func(_ context.Context, msg int) error {
		ch <- msg
		return nil
	}))

	// no skip errors
	assert.ErrorIs(t, topic.Emit(ctx, 1), assert.AnError)
	assert.Len(t, ch, 0)

	// skip errors
	assert.NoError(t, topic.Emit(ctx, 2, WithEmitSkipErrors()))
	assert.Equal(t, 2, <-ch)
}

func TestEventBus_Async(t *testing.T) {
	topic := NewEvent[int]()
	ch := make(chan int, 2)
	wg := sync.WaitGroup{}

	_ = topic.On(HandlerFunc[int](func(_ context.Context, msg int) error {
		defer wg.Done()
		time.Sleep(time.Millisecond * 100)
		ch <- 10 * msg
		return nil
	}))
	_ = topic.On(HandlerFunc[int](func(_ context.Context, msg int) error {
		defer wg.Done()
		time.Sleep(time.Millisecond * 100)
		ch <- 100 * msg
		return nil
	}))

	// sync
	wg.Add(2)
	start := time.Now()
	assert.NoError(t, topic.Emit(ctx, 1))
	wg.Wait()
	assert.Equal(t, 10, <-ch)
	assert.Equal(t, 100, <-ch)
	assert.Len(t, ch, 0)
	assert.GreaterOrEqual(t, time.Since(start).Milliseconds(), int64(200))

	// async
	wg.Add(2)
	start = time.Now()
	assert.NoError(t, topic.Emit(ctx, 2, WithEmitAsync()))
	wg.Wait()
	assert.Len(t, ch, 2)
	assert.Contains(t, []int{20, 200}, <-ch)
	assert.Contains(t, []int{20, 200}, <-ch)
	assert.GreaterOrEqual(t, time.Since(start).Milliseconds(), int64(100))
	assert.Less(t, time.Since(start).Milliseconds(), int64(150))
}
