module github.com/go-fries/fries/jsonrpc/v3

go 1.24.0

replace (
	github.com/go-fries/fries/codec/json/v3 => ../codec/json
	github.com/go-fries/fries/codec/v3 => ../codec
)

require (
	github.com/go-fries/fries/codec/json/v3 v3.10.0
	github.com/go-fries/fries/codec/v3 v3.10.0
	github.com/google/uuid v1.6.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/testify v1.11.1
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
