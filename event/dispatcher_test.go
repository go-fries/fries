package event

import (
	"context"
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
			dispatcher.Wait()
			assert.Equal(t, "123456", <-ch)
		}()

		assert.NoError(t, dispatcher.Dispatch(ctx, &OrderEvent{OrderID: "123456"}))
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
