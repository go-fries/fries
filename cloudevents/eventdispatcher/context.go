package eventdispatcher

import (
	"context"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

type contextKey struct{}

func NewContext(ctx context.Context, event cloudevents.Event) context.Context {
	return context.WithValue(ctx, contextKey{}, event)
}

func FromContext(ctx context.Context) (cloudevents.Event, bool) {
	event, ok := ctx.Value(contextKey{}).(cloudevents.Event)
	if !ok {
		return cloudevents.Event{}, false
	}
	return event, true
}
