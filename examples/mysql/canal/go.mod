module github.com/go-fries/fries/examples/mysql/canal/v3

go 1.24.0

replace (
	github.com/go-fries/fries/codec/json/v3 => ../../../codec/json
	github.com/go-fries/fries/codec/v3 => ../../../codec
	github.com/go-fries/fries/mysql/canal/positioner/redis/v3 => ./../../../mysql/canal/positioner/redis/
	github.com/go-fries/fries/mysql/canal/v3 => ../../../mysql/canal/
)

require (
	github.com/go-fries/fries/mysql/canal/positioner/redis/v3 v3.12.0
	github.com/go-fries/fries/mysql/canal/v3 v3.12.0
)

require (
	filippo.io/edwards25519 v1.1.1 // indirect
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-fries/fries/codec/json/v3 v3.12.0 // indirect
	github.com/go-fries/fries/codec/v3 v3.12.0 // indirect
	github.com/go-mysql-org/go-mysql v1.13.0 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.18.4 // indirect
	github.com/pingcap/errors v0.11.5-0.20251231075859-d18e03b1da26 // indirect
	github.com/pingcap/failpoint v0.0.0-20251231045439-91d91e123837 // indirect
	github.com/pingcap/log v1.1.1-0.20251231064424-c412c24f73b2 // indirect
	github.com/pingcap/tidb/pkg/parser v0.0.0-20260205124303-d537742bb2cd // indirect
	github.com/redis/go-redis/v9 v9.17.3 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.1 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
)
