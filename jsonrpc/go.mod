module github.com/go-fries/fries/jsonrpc/v4

go 1.25.0

replace (
	github.com/go-fries/fries/codec/json/v4 => ../codec/json
	github.com/go-fries/fries/codec/v4 => ../codec
)

require (
	github.com/go-fries/fries/codec/json/v4 v4.0.0
	github.com/go-fries/fries/codec/v4 v4.0.0
	github.com/google/uuid v1.6.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/testify v1.11.1
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
