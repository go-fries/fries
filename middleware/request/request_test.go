package request

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-fries/fries/v3/event"
)

var c = make(chan From, 4)

type listener struct {
	t *testing.T
}

var _ event.Listener = (*listener)(nil)

func (l *listener) Listen() []event.Event {
	return []event.Event{
		&BeforeEvent{},
		&AfterEvent{},
	}
}

func (l *listener) Handle(event event.Event) {
	switch e := event.(type) {
	case *BeforeEvent:
		assert.Equal(l.t, BeforeEvent{}, event.Event())
		assert.Equal(l.t, e.Req, "req")
		assert.Equal(l.t, e.Ctx, context.Background())
		c <- e.From
	case *AfterEvent:
		assert.Equal(l.t, AfterEvent{}, event.Event())
		assert.Equal(l.t, e.Req, "req")
		assert.Equal(l.t, e.Reply, "reply")
		assert.Nil(l.t, e.Err)
		assert.Equal(l.t, e.Ctx, context.Background())
		c <- e.From
	}
}

func TestRequest(t *testing.T) {
	var (
		d       = event.NewDispatcher()
		handler = func(_ context.Context, req any) (reply any, err error) {
			assert.Equal(t, "req", req)
			return "reply", nil
		}
	)

	d.AddListener(&listener{t: t})

	reply, err := Server(d)(handler)(context.Background(), "req")
	assert.Equal(t, "reply", reply)
	assert.Nil(t, err)
	assert.Equal(t, FromServer, <-c)
	assert.Equal(t, FromServer, <-c)

	reply, err = Client(d)(handler)(context.Background(), "req")
	assert.Equal(t, "reply", reply)
	assert.Nil(t, err)
	assert.Equal(t, FromClient, <-c)
	assert.Equal(t, FromClient, <-c)
}
