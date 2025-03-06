package event

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var ctx = context.Background()

func TestDispatcher(t *testing.T) {
	dispatcher := NewDispatcher()

	t.Run("Multiple event bindings based on callback functions", func(t *testing.T) {
		dispatcher.Reset()

		dispatcher.RegisterListeners(
			AdaptListener(ListenerFunc[*UserEvent](func(_ context.Context, event *UserEvent) error {
				t.Logf("this is user listener, the name is %s", event.Name)
				assert.Equal(t, "zhangsan", event.Name)
				return nil
			})),
			AdaptListenerFunc(func(_ context.Context, event *OrderEvent) error {
				t.Logf("this is order listener, the order id is %s", event.OrderID)
				assert.Equal(t, "123456", event.OrderID)
				return nil
			}),
		)

		assert.NoError(t, dispatcher.Dispatch(ctx, &UserEvent{Name: "zhangsan"}))
		assert.NoError(t, dispatcher.Dispatch(ctx, &OrderEvent{OrderID: "123456"}))
	})

	t.Run("Multiple event bindings based on callback functions", func(t *testing.T) {
		dispatcher.Reset()

		dispatcher.RegisterListeners(
			AdaptListener(ListenerFunc[*UserEvent](func(_ context.Context, event *UserEvent) error {
				t.Logf("this is user listener, the name is %s", event.Name)
				assert.Equal(t, "zhangsan", event.Name)
				return nil
			})),
			AdaptListener(ListenerFunc[*UserEvent](func(_ context.Context, event *UserEvent) error {
				t.Logf("this is user listener, the name is %s", event.Name)
				assert.Equal(t, "zhangsan", event.Name)
				return nil
			})),
		)

		assert.NoError(t, dispatcher.Dispatch(ctx, &UserEvent{Name: "zhangsan"}))
		assert.NoError(t, dispatcher.Dispatch(ctx, &OrderEvent{OrderID: "123456"}))
	})

	t.Run("Mixed event binding based on structures and callbacks", func(t *testing.T) {
		dispatcher.Reset()

		dispatcher.RegisterListeners(
			AdaptListener(ListenerFunc[*UserEvent](func(_ context.Context, event *UserEvent) error {
				t.Logf("this is user listener, the name is %s", event.Name)
				assert.Equal(t, "zhangsan", event.Name)
				return nil
			})),
			AdaptListener(&UserListener{tb: t}),
			AdaptListener(ListenerFunc[*UserEvent](func(_ context.Context, event *UserEvent) error {
				t.Logf("this is user listener, the name is %s", event.Name)
				assert.Equal(t, "zhangsan", event.Name)
				return nil
			})),
			AdaptListener(&OrderListener{tb: t}),
		)

		assert.NoError(t, dispatcher.Dispatch(ctx, &UserEvent{Name: "zhangsan"}))
		assert.NoError(t, dispatcher.Dispatch(ctx, &OrderEvent{OrderID: "123456"}))
	})

	t.Run("wait for all listeners to complete", func(t *testing.T) {
		defer dispatcher.Reset()

		ch := make(chan string, 1)

		dispatcher.RegisterListeners(
			AdaptListenerFunc(func(_ context.Context, event *OrderEvent) error {
				assert.Equal(t, "123456", event.OrderID)
				time.Sleep(1 * time.Second)
				ch <- event.OrderID
				return nil
			}),
		)

		go func() {
			assert.Equal(t, "123456", <-ch)
		}()

		assert.NoError(t, dispatcher.Dispatch(ctx, &OrderEvent{OrderID: "123456"}))
	})

	t.Run("has error when [withError] eq true", func(t *testing.T) {
		var l sync.Mutex
		ec := 0
		d := NewDispatcher(WithError(), WithParallel(1))
		d.RegisterListeners(
			AdaptListenerFunc(func(_ context.Context, _ *UserEvent) error {
				l.Lock()
				ec++
				l.Unlock()
				return errors.New("some error")
			}),
			AdaptListenerFunc(func(_ context.Context, _ *UserEvent) error {
				l.Lock()
				ec++
				l.Unlock()
				return errors.New("some error")
			}),
			AdaptListenerFunc(func(_ context.Context, _ *UserEvent) error {
				l.Lock()
				ec++
				l.Unlock()
				return nil
			}),
			AdaptListenerFunc(func(_ context.Context, _ *UserEvent) error {
				l.Lock()
				ec++
				l.Unlock()
				return nil
			}),
			AdaptListenerFunc(func(_ context.Context, _ *UserEvent) error {
				l.Lock()
				ec++
				l.Unlock()
				return nil
			}),
		)
		err := d.Dispatch(ctx, &UserEvent{})
		assert.Equal(t, 1, ec)
		assert.Error(t, err)
	})

	t.Run("has error when [withError] eq false", func(t *testing.T) {
		var l sync.Mutex
		ec := 0
		d := NewDispatcher(WithoutError(), WithParallel(3))
		d.RegisterListeners(
			AdaptListenerFunc(func(_ context.Context, _ *UserEvent) error {
				l.Lock()
				ec++
				l.Unlock()
				return errors.New("some error")
			}),
			AdaptListenerFunc(func(_ context.Context, _ *UserEvent) error {
				l.Lock()
				ec++
				l.Unlock()
				return errors.New("some error")
			}),
			AdaptListenerFunc(func(_ context.Context, _ *UserEvent) error {
				l.Lock()
				ec++
				l.Unlock()
				return nil
			}),
			AdaptListenerFunc(func(_ context.Context, _ *UserEvent) error {
				l.Lock()
				ec++
				l.Unlock()
				return nil
			}),
		)
		err := d.Dispatch(ctx, &UserEvent{})
		assert.Equal(t, 4, ec)
		assert.NoError(t, err)
	})

	t.Run("check the number of parallel goroutines", func(t *testing.T) {
		parallel := 3
		d := NewDispatcher(WithoutError(), WithParallel(parallel))
		startedCount := runtime.NumGoroutine()
		for i := 0; i < 10; i++ {
			d.RegisterListeners(
				AdaptListenerFunc(func(_ context.Context, _ *UserEvent) error {
					<-time.After(200 * time.Millisecond)
					return nil
				}))
		}
		go func() {
			<-time.After(300 * time.Millisecond)
			currentNum := runtime.NumGoroutine()
			// startedCount should eq currentNum - 1(this goroutine) - parallel
			assert.Equal(t, startedCount, currentNum-1-parallel)
		}()
		_ = d.Dispatch(ctx, &UserEvent{})
	})

	t.Run("Check if the runningOptions of the Dispatch method are valid", func(t *testing.T) {
		var l sync.Mutex
		ec := 0
		d := NewDispatcher(WithError(), WithParallel(1))
		d.RegisterListeners(
			AdaptListenerFunc(func(_ context.Context, _ *UserEvent) error {
				l.Lock()
				ec++
				l.Unlock()
				return errors.New("some error")
			}),
			AdaptListenerFunc(func(_ context.Context, _ *UserEvent) error {
				l.Lock()
				ec++
				l.Unlock()
				return nil
			}),
			AdaptListenerFunc(func(_ context.Context, _ *UserEvent) error {
				l.Lock()
				ec++
				l.Unlock()
				return nil
			}),
		)
		err := d.Dispatch(ctx, &UserEvent{}, WithRunningParallel(-1), WithoutRunningError())
		assert.Equal(t, 3, ec)
		assert.NoError(t, err)
	})
}

type UserEvent struct {
	Name string
}

type UserListener struct {
	tb testing.TB
}

func (u *UserListener) Handle(_ context.Context, event *UserEvent) error {
	assert.Equal(u.tb, "zhangsan", event.Name)
	return nil
}

type OrderEvent struct {
	OrderID string
}

type OrderListener struct {
	tb testing.TB
}

func (o *OrderListener) Listen() []string {
	return []string{"order"}
}

func (o *OrderListener) Handle(_ context.Context, event *OrderEvent) error {
	assert.Equal(o.tb, "123456", event.OrderID)
	return nil
}
