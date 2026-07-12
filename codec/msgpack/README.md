# MessagePack Codec

MessagePack support for `github.com/go-fries/fries/codec/v4`, implemented with
`github.com/vmihailenco/msgpack/v5`.

```go
c := msgpack.Codec{}

data, err := c.Marshal(value)
if err != nil {
    return err
}

err = c.Unmarshal(data, &value)
```

The zero value is ready to use and safe for concurrent use.
