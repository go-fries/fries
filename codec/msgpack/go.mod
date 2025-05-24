module github.com/go-fries/fries/codec/msgpack/v3

go 1.23.0

replace github.com/go-fries/fries/codec/v3 => ../

require (
	github.com/go-fries/fries/codec/v3 v3.3.0
	github.com/stretchr/testify v1.10.0
	github.com/vmihailenco/msgpack/v5 v5.4.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
