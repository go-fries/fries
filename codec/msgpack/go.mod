module github.com/go-fries/fries/codec/msgpack/v4

go 1.26.0

replace github.com/go-fries/fries/codec/v4 => ../

require (
	github.com/go-fries/fries/codec/v4 v4.1.0
	github.com/stretchr/testify v1.12.1
	github.com/vmihailenco/msgpack/v5 v5.4.1
)

require (
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
)
