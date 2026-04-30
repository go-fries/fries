module github.com/go-fries/fries/mysql/canal/positioner/redis/v3

go 1.25.0

replace (
	github.com/go-fries/fries/codec/json/v3 => ./../../../../codec/json
	github.com/go-fries/fries/codec/v3 => ./../../../../codec
	github.com/go-fries/fries/mysql/canal/v3 => ../../
)

require (
	github.com/go-fries/fries/codec/json/v3 v3.12.0
	github.com/go-fries/fries/codec/v3 v3.12.0
	github.com/go-fries/fries/mysql/canal/v3 v3.12.0
	github.com/go-mysql-org/go-mysql v1.13.0
	github.com/redis/go-redis/v9 v9.19.0
	github.com/stretchr/testify v1.11.1
)

require (
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/goccy/go-json v0.10.6 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/pingcap/errors v0.11.5-0.20260310054046-9c8b3586e4b2 // indirect
	github.com/pingcap/failpoint v0.0.0-20260406204437-bbc9d102c19e // indirect
	github.com/pingcap/log v1.1.1-0.20260227082333-572e590d08f1 // indirect
	github.com/pingcap/tidb/pkg/parser v0.0.0-20260401125306-84124227a1f2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/text v0.36.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
