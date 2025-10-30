package async

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type ChannelQueue struct {
	data chan *Message
}

var _ Queue = (*ChannelQueue)(nil)

func NewChannelQueue(size int) *ChannelQueue {
	return &ChannelQueue{
		data: make(chan *Message, size),
	}
}

func (c *ChannelQueue) Enqueue(ctx context.Context, message *Message) error {
	select {
	case c.data <- message:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *ChannelQueue) Dequeue(ctx context.Context) (*Message, error) {
	select {
	case msg := <-c.data:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestProvider(t *testing.T) {
	queue := NewChannelQueue(10)
	provider := NewProvider(queue)

	err := provider.Add("test_task", func(ctx context.Context, args any) error {
		t.Logf("Processing task with args: %v", args)
		return nil
	})
	require.NoError(t, err)

	go func() {
		err = provider.Start(t.Context())
		require.NoError(t, err)
	}()

	for {
		err = provider.Submit(t.Context(), "test_task", "sample_args")
		require.NoError(t, err)
	}
}
