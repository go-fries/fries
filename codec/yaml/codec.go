package yaml

import (
	"github.com/go-fries/fries/codec/v3"
	"gopkg.in/yaml.v3"
)

var Codec codec.Codec = &yamlCodec{}

// yamlCodec is a YAML codec.
type yamlCodec struct{}

// Marshal converts the given data into a byte slice using YAML.
func (c *yamlCodec) Marshal(data any) ([]byte, error) {
	return yaml.Marshal(data)
}

// Unmarshal converts the given byte slice into a data structure using YAML.
func (c *yamlCodec) Unmarshal(src []byte, dest any) error {
	return yaml.Unmarshal(src, dest)
}
