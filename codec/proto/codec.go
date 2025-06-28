package proto

import (
	"fmt"

	"github.com/go-fries/fries/codec/v3"
	"google.golang.org/protobuf/proto"
)

var Codec codec.Codec = &protoCodec{}

var ErrInvalidProtoMessage = fmt.Errorf("data must implement proto.Message interface")

// protoCodec is a Protocol Buffers codec.
type protoCodec struct{}

// Marshal converts the given data into a byte slice using Protocol Buffers.
func (c *protoCodec) Marshal(data any) ([]byte, error) {
	msg, ok := data.(proto.Message)
	if !ok {
		return nil, ErrInvalidProtoMessage
	}
	return proto.Marshal(msg)
}

// Unmarshal converts the given byte slice into a data structure using Protocol Buffers.
func (c *protoCodec) Unmarshal(src []byte, dest any) error {
	msg, ok := dest.(proto.Message)
	if !ok {
		return ErrInvalidProtoMessage
	}
	return proto.Unmarshal(src, msg)
}
