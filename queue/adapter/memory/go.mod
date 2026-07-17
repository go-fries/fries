module github.com/go-fries/fries/queue/adapter/memory/v4

go 1.25.0

replace (
	github.com/go-fries/fries/codec/v4 => ../../../codec
	github.com/go-fries/fries/queue/v4 => ../../
	github.com/go-fries/fries/retry/v4 => ../../../retry
)

require (
	github.com/go-fries/fries/queue/v4 v4.0.0-beta.3
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-fries/fries/codec/v4 v4.0.0-beta.3 // indirect
	github.com/go-fries/fries/retry/v4 v4.0.0-beta.3 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
