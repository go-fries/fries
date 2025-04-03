module github.com/go-fries/fries/examples/cache/v3

go 1.23.0

replace (
	github.com/go-fries/fries/cache/redis/v3 => ../../cache/redis/
	github.com/go-fries/fries/cache/v3 => ../../cache/
)

require (
	github.com/go-fries/fries/cache/redis/v3 v3.0.0-rc.2
	github.com/go-fries/fries/cache/v3 v3.0.0-rc.2
	github.com/redis/go-redis/v9 v9.7.3
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-fries/fries/codec/json/v3 v3.0.0-rc.2 // indirect
	github.com/go-fries/fries/codec/v3 v3.0.0-rc.2 // indirect
	github.com/go-fries/fries/locker/redis/v3 v3.0.0-rc.2 // indirect
	github.com/go-fries/fries/locker/v3 v3.0.0-rc.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
)
