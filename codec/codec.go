package codec

// Codec is an interface for serialization and deserialization operations.
type Codec interface {
	// Marshal converts the given data into a byte slice.
	// This method supports converting data structures of any type into a byte slice for transmission or storage.
	// Parameters:
	//   data - The data to be serialized, of any type.
	// Return value:
	//   It returns the byte slice obtained after serialization, and an error if serialization fails.
	Marshal(data any) ([]byte, error)

	// Unmarshal converts the given byte slice into a data structure.
	// This method supports deserializing a byte slice into a data structure of any type.
	// Parameters:
	//   src   - The byte slice to be deserialized.
	//   dest  - A pointer to the destination data structure where the deserialized data will be stored.
	//            The specific type of the data structure needs to match the content of the byte slice.
	// Return value:
	//   It returns an error if deserialization fails.
	Unmarshal(src []byte, dest any) error
}
