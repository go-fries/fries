module github.com/go-fries/fries/jsonrpc/v3

go 1.25.0

replace (
	github.com/go-fries/fries/codec/json/v3 => ../codec/json
	github.com/go-fries/fries/codec/v3 => ../codec
)

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/go-fries/fries/codec/json/v3 v3.0.0-00010101000000-000000000000
	github.com/go-fries/fries/codec/v3 v3.9.2
	github.com/google/uuid v1.6.0
)
