package event

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var ch = make(chan string, 1)

type testEvent struct {
	Payload string
}

func (e *testEvent) Event() any {
	return testEvent{}
}

type test2Event struct {
	Payload string
}

func (e *test2Event) Event() any {
	return test2Event{}
}

type testListener struct{}

func newTestListener() *testListener {
	return &testListener{}
}

func (l *testListener) Listen() []Event {
	return []Event{
		&testEvent{},
		&test2Event{},
	}
}

func (l *testListener) Handle(event Event) {
	if e, ok := event.(*testEvent); ok {
		ch <- e.Payload
	}
	panic("invalid data")
}

func TestDispatcher(t *testing.T) {
	d := NewDispatcher(
		WithRecovery(func(err any, _ Listener, _ Event) {
			assert.Equal(t, "invalid data", err.(string))
		}),
	)

	d.AddListener(newTestListener())

	d.Dispatch(&testEvent{
		Payload: "123",
	})
	assert.Equal(t, "123", <-ch)

	assert.NotPanics(t, func() {
		d.Dispatch(&test2Event{})
	})
	assert.Len(t, ch, 0)

	d.DispatchAsync(&testEvent{
		Payload: "345",
	})
	d.Wait()
	assert.Equal(t, "345", <-ch)
}
