# XML Codec

XML support for `github.com/go-fries/fries/codec/v4`, implemented with the Go
standard library's `encoding/xml` package.

```go
c := xml.Codec{}

data, err := c.Marshal(value)
if err != nil {
    return err
}

err = c.Unmarshal(data, &value)
```

The zero value is ready to use and safe for concurrent use.
