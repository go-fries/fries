package proto_test

import (
	"testing"

	"github.com/go-fries/fries/codec/proto/v4"
	testproto "github.com/go-fries/fries/codec/proto/v4/internal/proto"
	"github.com/go-fries/fries/codec/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodec(t *testing.T) {
	c := proto.Codec{}
	assert.Implements(t, (*codec.Codec)(nil), c)

	want := &testproto.TestMessage{Name: "test", Value: 123}
	data, err := c.Marshal(want)
	require.NoError(t, err)

	got := &testproto.TestMessage{}
	require.NoError(t, c.Unmarshal(data, got))
	assert.Equal(t, want.GetName(), got.GetName())
	assert.Equal(t, want.GetValue(), got.GetValue())
}

func TestCodecInvalidMessage(t *testing.T) {
	c := proto.Codec{}

	_, err := c.Marshal("invalid")
	assert.ErrorIs(t, err, proto.ErrInvalidMessage)
	assert.ErrorIs(t, c.Unmarshal(nil, new(string)), proto.ErrInvalidMessage)
}

func TestCodecInvalidData(t *testing.T) {
	c := proto.Codec{}
	err := c.Unmarshal([]byte{0xff}, &testproto.TestMessage{})
	assert.Error(t, err)
}
