# YAML Codec

YAML support for `github.com/go-fries/fries/codec/v4`, implemented with
`gopkg.in/yaml.v3`.

```go
c := yaml.Codec{}

data, err := c.Marshal(value)
if err != nil {
    return err
}

err = c.Unmarshal(data, &value)
```

The zero value is ready to use and safe for concurrent use.
