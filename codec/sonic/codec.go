package sonic

import (
	"github.com/bytedance/sonic"
	"github.com/go-fries/fries/codec/v4"
)

// Codec encodes and decodes JSON values using Sonic. Its zero value is ready to use.
type Codec struct{}

var _ codec.Codec = Codec{}

// Marshal encodes data as JSON.
func (Codec) Marshal(data any) ([]byte, error) {
	return sonic.Marshal(data)
}

// Unmarshal decodes JSON data into dest.
func (Codec) Unmarshal(src []byte, dest any) error {
	return sonic.Unmarshal(src, dest)
}
