package queue

import (
	"encoding/json"

	"github.com/go-fries/fries/codec/v3"
)

var defaultCodec codec.Codec = jsonCodec{}

type jsonCodec struct{}

func (jsonCodec) Marshal(data any) ([]byte, error) {
	return json.Marshal(data)
}

func (jsonCodec) Unmarshal(src []byte, dest any) error {
	return json.Unmarshal(src, dest)
}
