package json

import (
	"encoding/json"

	"github.com/go-fries/fries/codec/v4"
)

// Codec encodes and decodes JSON values. Its zero value is ready to use.
type Codec struct{}

var _ codec.Codec = Codec{}

// Marshal encodes data as JSON.
func (Codec) Marshal(data any) ([]byte, error) {
	return json.Marshal(data)
}

// Unmarshal decodes JSON data into dest.
func (Codec) Unmarshal(src []byte, dest any) error {
	return json.Unmarshal(src, dest)
}
