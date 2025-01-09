package msgpack

import (
	"github.com/go-fries/fries/codec/v3"
	"github.com/vmihailenco/msgpack/v5"
)

var Codec codec.Codec = &msgPackCodec{}

type msgPackCodec struct{}

func (j *msgPackCodec) Marshal(data any) ([]byte, error) {
	return msgpack.Marshal(data)
}

func (j *msgPackCodec) Unmarshal(src []byte, dest any) error {
	return msgpack.Unmarshal(src, dest)
}
