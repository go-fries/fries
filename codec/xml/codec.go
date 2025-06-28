package xml

import (
	"encoding/xml"

	"github.com/go-fries/fries/codec/v3"
)

var Codec codec.Codec = &xmlCodec{}

// codec is an XML codec.
type xmlCodec struct{}

var _ codec.Codec = (*xmlCodec)(nil)

// Marshal converts the given data into a byte slice using XML.
func (c *xmlCodec) Marshal(data any) ([]byte, error) {
	return xml.Marshal(data)
}

// Unmarshal converts the given byte slice into a data structure using XML.
func (c *xmlCodec) Unmarshal(src []byte, dest any) error {
	return xml.Unmarshal(src, dest)
}
