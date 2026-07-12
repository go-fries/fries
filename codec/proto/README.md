# Protocol Buffers Codec

Protocol Buffers support for `github.com/go-fries/fries/codec/v4`, implemented
with `google.golang.org/protobuf/proto`.

```go
c := proto.Codec{}

data, err := c.Marshal(message)
if err != nil {
    return err
}

err = c.Unmarshal(data, destination)
```

Values passed to `Marshal` and `Unmarshal` must implement
`google.golang.org/protobuf/proto.Message`. The zero codec value is ready to use
and safe for concurrent use.
