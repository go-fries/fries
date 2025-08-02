module github.com/go-fries/fries/mysql/canal/positioner/redis/v3

go 1.23.0

replace (
	github.com/go-fries/fries/codec/json/v3 => ./../../../../codec/json
	github.com/go-fries/fries/codec/v3 => ./../../../../codec
	github.com/go-fries/fries/mysql/canal/v3 => ../../
)

require (
	github.com/go-fries/fries/codec/json/v3 v3.7.1
	github.com/go-fries/fries/codec/v3 v3.7.1
	github.com/go-fries/fries/mysql/canal/v3 v3.7.1
	github.com/go-mysql-org/go-mysql v1.12.0
	github.com/redis/go-redis/v9 v9.11.0
	github.com/stretchr/testify v1.10.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/BurntSushi/toml v1.5.0 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/pingcap/errors v0.11.5-0.20250523034308-74f78ae071ee // indirect
	github.com/pingcap/failpoint v0.0.0-20240528011301-b51a646c7c86 // indirect
	github.com/pingcap/log v1.1.1-0.20250514022801-14f3b4ca066e // indirect
	github.com/pingcap/tidb/pkg/parser v0.0.0-20250801151445-f13696c16424 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
