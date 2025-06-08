module github.com/go-fries/fries/queue/redis/v3

go 1.23.0

replace github.com/go-fries/fries/queue/v3 => ../

require (
	github.com/go-fries/fries/queue/v3 v3.0.0-00010101000000-000000000000
	github.com/redis/go-redis/v9 v9.10.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
)
