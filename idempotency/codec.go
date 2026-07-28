package idempotency

import "encoding/json"

// Codec marshals and unmarshals values persisted by DoValue.
//
// Implementations of github.com/go-fries/fries/codec/v4.Codec satisfy this
// interface without an adapter.
type Codec interface {
	Marshal(any) ([]byte, error)
	Unmarshal([]byte, any) error
}

type jsonCodec struct{}

func (jsonCodec) Marshal(value any) ([]byte, error) {
	return json.Marshal(value)
}

func (jsonCodec) Unmarshal(data []byte, value any) error {
	return json.Unmarshal(data, value)
}
