module github.com/go-fries/fries/queue/redis/v3

go 1.25.0

replace github.com/go-fries/fries/queue/v3 => ../

require (
	github.com/go-fries/fries/queue/v3 v3.14.0
	github.com/redis/go-redis/v9 v9.20.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
)
