package sonic

import (
	"github.com/bytedance/sonic"
	"github.com/go-fries/fries/codec/v3"
)

var Codec codec.Codec = &sonicCodec{}

type sonicCodec struct{}

func (j *sonicCodec) Marshal(data any) ([]byte, error) {
	return sonic.Marshal(data)
}

func (j *sonicCodec) Unmarshal(src []byte, dest any) error {
	return sonic.Unmarshal(src, dest)
}
