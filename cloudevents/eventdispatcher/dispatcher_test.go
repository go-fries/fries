package eventdispatcher

import (
	"context"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/assert"
)

type TestEvent struct {
	User string
}

type TestListener struct {
	t *testing.T
}

var _ AnyListener[TestEvent] = (*TestListener)(nil)

func (l *TestListener) Handle(ctx context.Context, event TestEvent) error {
	assert.Equal(l.t, "test-user", event.User)
	return nil
}

func TestDispatcher(t *testing.T) {
	dispatcher := NewDispatcher()

	t.Run("Register and Dispatch", func(t *testing.T) {
		defer dispatcher.Reset()

		eventType := "test.event" //nolint:goconst
		listener := ListenerFunc[struct{}](func(ctx context.Context, event struct{}) error {
			return nil // Simulate handling the event
		})

		dispatcher.AddListener(eventType, listener)

		event := cloudevents.NewEvent()
		event.SetID("test-event")
		event.SetType(eventType)

		assert.NoError(t, dispatcher.Dispatch(t.Context(), event))
	})

	t.Run("Dispatch without listeners", func(t *testing.T) {
		defer dispatcher.Reset()

		event := cloudevents.NewEvent()
		event.SetID("test-event")
		event.SetType("non.existent.event")

		err := dispatcher.Dispatch(t.Context(), event)
		assert.Equal(t, ErrNoListener, err)
	})

	t.Run("Dispatch with multiple listeners", func(t *testing.T) {
		defer dispatcher.Reset()

		eventType := "test.event"
		listener1 := ListenerFunc[struct{}](func(ctx context.Context, event struct{}) error {
			return nil // Simulate handling the event
		})
		listener2 := ListenerFunc[struct{}](func(ctx context.Context, event struct{}) error {
			return nil // Simulate handling the event
		})

		dispatcher.AddListener(eventType, listener1)
		dispatcher.AddListener(eventType, listener2)

		event := cloudevents.NewEvent()
		event.SetID("test-event")
		event.SetType(eventType)

		assert.NoError(t, dispatcher.Dispatch(t.Context(), event))
	})

	t.Run("Dispatch with unmarshal error", func(t *testing.T) {
		defer dispatcher.Reset()

		eventType := "test.event"
		listener := ListenerFunc[struct{}](func(ctx context.Context, event struct{}) error {
			return nil // Simulate handling the event
		})

		dispatcher.AddListener(eventType, listener)

		event := cloudevents.NewEvent()
		event.SetID("test-event")
		event.SetType(eventType)
		assert.NoError(t, event.SetData(cloudevents.ApplicationJSON, "invalid data"))

		err := dispatcher.Dispatch(t.Context(), event)
		assert.Equal(t, ErrFailedToUnmarshalEvent, err)
	})

	t.Run("Dispatch with context", func(t *testing.T) {
		defer dispatcher.Reset()

		eventType := "test.event"
		listener := ListenerFunc[struct{}](func(ctx context.Context, event struct{}) error {
			assert.Equal(t, "value", ctx.Value("key"))
			evt, ok := FromContext(ctx)
			assert.True(t, ok)
			assert.Equal(t, "test-event", evt.ID())
			return nil
		})

		dispatcher.AddListener(eventType, listener)

		event := cloudevents.NewEvent()
		event.SetID("test-event")
		event.SetType(eventType)

		ctx := context.WithValue(t.Context(), "key", "value") //nolint:staticcheck

		assert.NoError(t, dispatcher.Dispatch(ctx, event))
	})

	t.Run("Dispatch custom event type", func(t *testing.T) {
		defer dispatcher.Reset()

		eventType := "custom.event"
		listener := ListenerFunc[string](func(ctx context.Context, event string) error {
			assert.Equal(t, "test data", event)
			return nil // Simulate handling the event
		})

		dispatcher.AddListener(eventType, listener)

		event := cloudevents.NewEvent()
		event.SetID("test-event")
		event.SetType(eventType)
		assert.NoError(t, event.SetData(cloudevents.TextPlain, "test data"))

		assert.NoError(t, dispatcher.Dispatch(t.Context(), event))
	})

	t.Run("Dispatch with listener adapter", func(t *testing.T) {
		defer dispatcher.Reset()

		eventType := "adapted.event"

		dispatcher.AddListener(eventType, ListenerAdapter(&TestListener{t}))

		event := cloudevents.NewEvent()
		event.SetID("test-event")
		event.SetType(eventType)
		assert.NoError(t, event.SetData(cloudevents.ApplicationJSON, TestEvent{User: "test-user"}))

		assert.NoError(t, dispatcher.Dispatch(t.Context(), event))

		// unmarshalling error
		unmarshalEvent := cloudevents.NewEvent()
		unmarshalEvent.SetID("test-event-unmarshal")
		unmarshalEvent.SetType(eventType)
		assert.NoError(t, unmarshalEvent.SetData(cloudevents.ApplicationJSON, "invalid data"))
		err := dispatcher.Dispatch(t.Context(), unmarshalEvent)
		assert.Equal(t, ErrFailedToUnmarshalEvent, err)
	})

	t.Run("Reset dispatcher", func(t *testing.T) {
		defer dispatcher.Reset()

		eventType := "test.event"
		listener := ListenerFunc[struct{}](func(ctx context.Context, event struct{}) error {
			return nil // Simulate handling the event
		})

		dispatcher.AddListener(eventType, listener)

		event := cloudevents.NewEvent()
		event.SetID("test-event")
		event.SetType(eventType)

		assert.NoError(t, dispatcher.Dispatch(t.Context(), event))

		dispatcher.Reset()

		err := dispatcher.Dispatch(t.Context(), event)
		assert.Equal(t, ErrNoListener, err)
	})
}
