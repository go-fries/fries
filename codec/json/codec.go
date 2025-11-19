package json //nolint:revive

import (
	"encoding/json"

	"github.com/go-fries/fries/codec/v3"
)

var Codec codec.Codec = &jsonCodec{}

type jsonCodec struct{}

func (j *jsonCodec) Marshal(data any) ([]byte, error) {
	return json.Marshal(data)
}

func (j *jsonCodec) Unmarshal(src []byte, dest any) error {
	return json.Unmarshal(src, dest)
}
