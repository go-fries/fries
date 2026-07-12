package msgpack

import (
	"github.com/go-fries/fries/codec/v4"
	"github.com/vmihailenco/msgpack/v5"
)

// Codec encodes and decodes MessagePack values. Its zero value is ready to use.
type Codec struct{}

var _ codec.Codec = Codec{}

// Marshal encodes data as MessagePack.
func (Codec) Marshal(data any) ([]byte, error) {
	return msgpack.Marshal(data)
}

// Unmarshal decodes MessagePack data into dest.
func (Codec) Unmarshal(src []byte, dest any) error {
	return msgpack.Unmarshal(src, dest)
}
