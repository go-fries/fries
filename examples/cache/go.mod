module github.com/go-fries/fries/examples/cache/v3

go 1.24.0

replace (
	github.com/go-fries/fries/cache/redis/v3 => ../../cache/redis/
	github.com/go-fries/fries/cache/v3 => ../../cache/
	github.com/go-fries/fries/codec/json/v3 => ../../codec/json/
	github.com/go-fries/fries/codec/v3 => ../../codec/
	github.com/go-fries/fries/locker/redis/v3 => ../../locker/redis/
	github.com/go-fries/fries/locker/v3 => ../../locker/
)

require (
	github.com/go-fries/fries/cache/redis/v3 v3.12.0
	github.com/go-fries/fries/cache/v3 v3.12.0
	github.com/redis/go-redis/v9 v9.18.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-fries/fries/codec/json/v3 v3.12.0 // indirect
	github.com/go-fries/fries/codec/v3 v3.12.0 // indirect
	github.com/go-fries/fries/locker/redis/v3 v3.12.0 // indirect
	github.com/go-fries/fries/locker/v3 v3.12.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
)
