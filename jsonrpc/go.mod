module github.com/go-fries/fries/jsonrpc/v3

go 1.24.0

replace (
	github.com/go-fries/fries/codec/json/v3 => ../codec/json
	github.com/go-fries/fries/codec/v3 => ../codec
)

require (
	github.com/go-fries/fries/codec/json/v3 v3.9.2
	github.com/go-fries/fries/codec/v3 v3.9.2
	github.com/google/uuid v1.6.0
)
