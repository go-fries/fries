module github.com/go-fries/fries/queue/v3

go 1.23.0

replace (
	github.com/go-fries/fries/codec/json/v3 => ../codec/json
	github.com/go-fries/fries/codec/v3 => ../codec
)

require (
	github.com/go-fries/fries/codec/json/v3 v3.0.0-00010101000000-000000000000
	github.com/go-fries/fries/codec/v3 v3.2.0
	github.com/google/uuid v1.6.0
	github.com/redis/go-redis/v9 v9.8.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
)
