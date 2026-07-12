package xml

import (
	"encoding/xml"

	"github.com/go-fries/fries/codec/v4"
)

// Codec encodes and decodes XML values. Its zero value is ready to use.
type Codec struct{}

var _ codec.Codec = Codec{}

// Marshal encodes data as XML.
func (Codec) Marshal(data any) ([]byte, error) {
	return xml.Marshal(data)
}

// Unmarshal decodes XML data into dest.
func (Codec) Unmarshal(src []byte, dest any) error {
	return xml.Unmarshal(src, dest)
}
