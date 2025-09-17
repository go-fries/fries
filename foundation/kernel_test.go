package foundation

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type (
	contextKey1 struct{}
	contextKey2 struct{}
)

type provider struct {
	t     *testing.T
	key   any
	value string
}

func newProvider(t *testing.T, key any, value string) *provider {
	return &provider{
		t:     t,
		key:   key,
		value: value,
	}
}

func (p *provider) Bootstrap(ctx context.Context) (context.Context, error) {
	return context.WithValue(ctx, p.key, p.value), nil
}

func (p *provider) Terminate(ctx context.Context) (context.Context, error) {
	v, ok := ctx.Value(p.key).(string)
	assert.True(p.t, ok)
	assert.Equal(p.t, p.value, v)

	return ctx, nil
}

func TestKernel(t *testing.T) {
	ch := make(chan string, 1)
	k := NewKernel(
		WithHandler(HandlerFunc(func(ctx context.Context) error {
			v1, ok := ctx.Value(contextKey1{}).(string)
			assert.True(t, ok)
			assert.Equal(t, "value1", v1)

			v2, ok := ctx.Value(contextKey2{}).(string)
			assert.True(t, ok)
			assert.Equal(t, "value2", v2)

			ch <- "done"

			return nil
		})),
		WithProviders(
			newProvider(t, contextKey1{}, "value1"),
		),
		WithTerminateTimeout(time.Second*10),
	)
	k.Register(
		newProvider(t, contextKey2{}, "value2"),
	)
	assert.NoError(t, k.Run(t.Context()))
	assert.Equal(t, "done", <-ch)
}
