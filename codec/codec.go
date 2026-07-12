package codec

// Codec marshals and unmarshals values.
type Codec interface {
	// Marshal encodes data.
	Marshal(data any) ([]byte, error)

	// Unmarshal decodes src into dest. Dest must be a value accepted by the
	// concrete codec, typically a non-nil pointer.
	Unmarshal(src []byte, dest any) error
}
