# Codec

`codec` defines the shared contract used by Fries components that serialize and
deserialize values.

```go
type Codec interface {
    Marshal(data any) ([]byte, error)
    Unmarshal(src []byte, dest any) error
}
```

Applications can depend on this interface without coupling business code to a
specific data format:

```go
func store(c codec.Codec, value any) ([]byte, error) {
    return c.Marshal(value)
}
```

Concrete implementations are published as separate modules:

- `codec/json`
- `codec/msgpack`
- `codec/proto`
- `codec/sonic`
- `codec/xml`
- `codec/yaml`

Custom codecs only need to implement the two interface methods.
