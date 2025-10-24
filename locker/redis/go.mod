module github.com/go-fries/fries/locker/redis/v3

go 1.24.0

replace github.com/go-fries/fries/locker/v3 => ../

require (
	github.com/go-fries/fries/locker/v3 v3.10.0
	github.com/google/uuid v1.6.0
	github.com/redis/go-redis/v9 v9.16.0
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
