package proto

import (
	"errors"

	"github.com/go-fries/fries/codec/v4"
	"google.golang.org/protobuf/proto"
)

// ErrInvalidMessage is returned when a value does not implement proto.Message.
var ErrInvalidMessage = errors.New("codec/proto: value must implement proto.Message")

// Codec encodes and decodes Protocol Buffers messages. Its zero value is ready to use.
type Codec struct{}

var _ codec.Codec = Codec{}

// Marshal encodes data as a Protocol Buffers message.
func (Codec) Marshal(data any) ([]byte, error) {
	msg, ok := data.(proto.Message)
	if !ok {
		return nil, ErrInvalidMessage
	}
	return proto.Marshal(msg)
}

// Unmarshal decodes a Protocol Buffers message into dest.
func (Codec) Unmarshal(src []byte, dest any) error {
	msg, ok := dest.(proto.Message)
	if !ok {
		return ErrInvalidMessage
	}
	return proto.Unmarshal(src, msg)
}
