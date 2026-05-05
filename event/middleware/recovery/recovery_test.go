package recovery

import (
	"context"
	"testing"

	"github.com/go-fries/fries/event/v3"
	"github.com/stretchr/testify/assert"
)

var ctx = context.Background()

type userEvent struct {
	Name string
}

func TestMiddleware(t *testing.T) {
	// t.Run("panic", func(t *testing.T) {
	// 	dispatcher := event.NewDispatcher()
	//
	// 	dispatcher.RegisterListeners(
	// 		event.AdaptListenerFunc(func(ctx context.Context, event userEvent) error {
	// 			panic("panic")
	// 		}),
	// 	)
	//
	// 	assert.Panics(t, func() {
	// 		_ = dispatcher.Dispatch(ctx, userEvent{
	// 			Name: "test",
	// 		})
	// 	})
	// })

	t.Run("recovery", func(t *testing.T) {
		dispatcher := event.NewDispatcher()
		dispatcher.Use(New())

		dispatcher.RegisterListeners(
			event.AdaptListenerFunc(func(context.Context, userEvent) error {
				panic("panic")
			}),
		)

		assert.NotPanics(t, func() {
			assert.NoError(t, dispatcher.Dispatch(ctx, userEvent{
				Name: "test",
			}))
		})
	})

	t.Run("recovery with custom handler", func(t *testing.T) {
		dispatcher := event.NewDispatcher()
		dispatcher.Use(New(WithHandler(func(_ context.Context, event, recovery any, stack []byte) {
			assert.Equal(t, "test", event.(userEvent).Name)
			assert.Equal(t, "panic", recovery.(string))
			assert.Contains(t, string(stack), "recovery_test.go")
		})))

		dispatcher.RegisterListeners(
			event.AdaptListenerFunc(func(context.Context, userEvent) error {
				panic("panic")
			}),
		)

		assert.NotPanics(t, func() {
			_ = dispatcher.Dispatch(ctx, userEvent{
				Name: "test",
			})
		})
	})
}
