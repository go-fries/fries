package yaml

import (
	"github.com/go-fries/fries/codec/v4"
	"gopkg.in/yaml.v3"
)

// Codec encodes and decodes YAML values. Its zero value is ready to use.
type Codec struct{}

var _ codec.Codec = Codec{}

// Marshal encodes data as YAML.
func (Codec) Marshal(data any) ([]byte, error) {
	return yaml.Marshal(data)
}

// Unmarshal decodes YAML data into dest.
func (Codec) Unmarshal(src []byte, dest any) error {
	return yaml.Unmarshal(src, dest)
}
