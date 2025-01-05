package request

import (
	"context"

	"github.com/go-kratos/kratos/v2/middleware"

	"github.com/go-fries/fries/v3/event"
)

const (
	BeforeName = "request.before"
	AfterName  = "request.after"
)

type From string

const (
	FromClient From = "client"
	FromServer From = "server"
)

type BeforeEvent struct {
	Ctx  context.Context
	Req  any
	From From
}

func (b *BeforeEvent) Event() any {
	return BeforeEvent{}
}

type AfterEvent struct {
	Ctx   context.Context
	Req   any
	Reply any
	Err   error
	From  From
}

func (a *AfterEvent) Event() any {
	return AfterEvent{}
}

func Server(d *event.Dispatcher) middleware.Middleware {
	return newMiddleware(d, FromServer)
}

func Client(d *event.Dispatcher) middleware.Middleware {
	return newMiddleware(d, FromClient)
}

func newMiddleware(d *event.Dispatcher, from From) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			if d != nil {
				d.Dispatch(&BeforeEvent{
					Ctx: ctx, Req: req, From: from,
				})
			}

			reply, err = handler(ctx, req)

			if d != nil {
				d.Dispatch(&AfterEvent{
					Ctx: ctx, Req: req, Reply: reply, Err: err, From: from,
				})
			}

			return
		}
	}
}
