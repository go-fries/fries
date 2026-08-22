module github.com/go-fries/fries/idempotency/v4

go 1.26.0

require (
	github.com/go-fries/fries/codec/json/v4 v4.1.0
	github.com/go-fries/fries/codec/v4 v4.1.0
	github.com/stretchr/testify v1.12.1
)

require go.yaml.in/yaml/v3 v3.0.5 // indirect

replace (
	github.com/go-fries/fries/codec/json/v4 => ../codec/json
	github.com/go-fries/fries/codec/v4 => ../codec
)
