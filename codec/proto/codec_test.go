package proto

import (
	"testing"

	"github.com/flc/go-fries/fries/codec/proto/v3/internal/proto"
	"github.com/stretchr/testify/assert"
)

func TestCodec(t *testing.T) {
	msg := &proto.TestMessage{
		Name:  "test",
		Value: 123,
	}

	data, err := Codec.Marshal(msg)
	assert.NoError(t, err)

	newMsg := &proto.TestMessage{}
	err = Codec.Unmarshal(data, newMsg)
	assert.NoError(t, err)

	assert.Equal(t, msg.Name, newMsg.Name)
	assert.Equal(t, msg.Value, newMsg.Value)
}

func TestCodec_ErrInvalidProtoMessage(t *testing.T) {
	_, err := Codec.Marshal("invalid data")
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidProtoMessage, err)

	var newMsg string
	err = Codec.Unmarshal([]byte("invalid data"), &newMsg)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidProtoMessage, err)
}
