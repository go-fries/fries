module github.com/go-fries/fries/idempotency/redis/v4

go 1.26.0

require (
	github.com/go-fries/fries/idempotency/v4 v4.0.0
	github.com/redis/go-redis/v9 v9.22.0
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-fries/fries/codec/json/v4 v4.0.0 // indirect
	github.com/go-fries/fries/codec/v4 v4.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/go-fries/fries/codec/json/v4 => ../../codec/json
	github.com/go-fries/fries/codec/v4 => ../../codec
	github.com/go-fries/fries/idempotency/v4 => ../
)
