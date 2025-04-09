module github.com/go-fries/fries/examples/mysql/canal/v3

go 1.23.0

replace (
	github.com/go-fries/fries/mysql/canal/redispositioner/v3 => ../../../mysql/canal/redispositioner/
	github.com/go-fries/fries/mysql/canal/v3 => ../../../mysql/canal/
)

require (
	github.com/go-fries/fries/mysql/canal/redispositioner/v3 v3.0.0-rc.2
	github.com/go-fries/fries/mysql/canal/v3 v3.0.0-rc.2
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/BurntSushi/toml v1.5.0 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-fries/fries/codec/json/v3 v3.0.0-rc.2 // indirect
	github.com/go-fries/fries/codec/v3 v3.0.0-rc.2 // indirect
	github.com/go-mysql-org/go-mysql v1.12.0 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/pingcap/errors v0.11.5-0.20250318082626-8f80e5cb09ec // indirect
	github.com/pingcap/failpoint v0.0.0-20240528011301-b51a646c7c86 // indirect
	github.com/pingcap/log v1.1.1-0.20241212030209-7e3ff8601a2a // indirect
	github.com/pingcap/tidb/pkg/parser v0.0.0-20250409042137-f09d70e33098 // indirect
	github.com/redis/go-redis/v9 v9.7.3 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
)
