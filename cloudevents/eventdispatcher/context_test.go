package eventdispatcher

import (
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	t.Run("with event", func(t *testing.T) {
		event := cloudevents.NewEvent()
		event.SetID("test-event")
		event.SetType("test.type")

		ctx := NewContext(t.Context(), event)

		retrievedEvent, ok := FromContext(ctx)
		assert.True(t, ok)
		assert.Equal(t, "test-event", retrievedEvent.ID())
		assert.Equal(t, event, retrievedEvent)
	})

	t.Run("empty context", func(t *testing.T) {
		emptyCtx := t.Context()
		_, ok := FromContext(emptyCtx)
		assert.False(t, ok)
	})
}
