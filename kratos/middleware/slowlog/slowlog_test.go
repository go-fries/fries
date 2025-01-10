package slowlog

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/stretchr/testify/assert"
)

type logger struct {
	t *testing.T
}

var _ log.Logger = (*logger)(nil)

var _ transport.Transporter = (*Transport)(nil)

type Transport struct {
	kind      transport.Kind
	endpoint  string
	operation string
}

func (tr *Transport) Kind() transport.Kind {
	return tr.kind
}

func (tr *Transport) Endpoint() string {
	return tr.endpoint
}

func (tr *Transport) Operation() string {
	return tr.operation
}

func (tr *Transport) RequestHeader() transport.Header {
	return nil
}

func (tr *Transport) ReplyHeader() transport.Header {
	return nil
}

func TestHTTP(t *testing.T) {
	err := errors.New("reply.error")
	logger := &logger{t}

	tests := []struct {
		name string
		kind middleware.Middleware
		err  error
		ctx  context.Context
	}{
		{
			"http-server@slow",
			Server(logger, WithThreshold(300*time.Millisecond)),
			err,
			func() context.Context {
				return transport.NewServerContext(
					context.Background(),
					&Transport{kind: transport.KindHTTP, endpoint: "endpoint", operation: "/package.service/method"},
				)
			}(),
		},
		{
			"http-server@notmal",
			Server(logger, WithThreshold(700*time.Millisecond)),
			nil,
			func() context.Context {
				return transport.NewServerContext(
					context.Background(),
					&Transport{kind: transport.KindHTTP, endpoint: "endpoint", operation: "/package.service/method"},
				)
			}(),
		},
		{
			"http-client@slow",
			Client(logger, WithThreshold(300*time.Millisecond)),
			nil,
			func() context.Context {
				return transport.NewClientContext(
					context.Background(),
					&Transport{kind: transport.KindHTTP, endpoint: "endpoint", operation: "/package.service/method"},
				)
			}(),
		},
		{
			"http-client@notmal",
			Client(logger, WithThreshold(700*time.Millisecond)),
			err,
			func() context.Context {
				return transport.NewClientContext(
					context.Background(),
					&Transport{kind: transport.KindHTTP, endpoint: "endpoint", operation: "/package.service/method"},
				)
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := tt.kind
			mockHandler := func(_ context.Context, _ any) (any, error) {
				time.Sleep(500 * time.Millisecond) // Simulate a slow request
				return "server response", tt.err
			}

			handler := mw(mockHandler)
			req := "request"
			resp, err := handler(tt.ctx, req)
			assert.Equal(t, tt.err, err)
			assert.Equal(t, "server response", resp)
		})
	}
}

func (l *logger) Log(_ log.Level, keyvals ...any) error {
	assert.Equal(l.t, "slowlog", keyvals[3])
	assert.Equal(l.t, "http", keyvals[5])
	assert.Equal(l.t, "/package.service/method", keyvals[7])
	assert.Equal(l.t, "request", keyvals[9])
	return nil
}
