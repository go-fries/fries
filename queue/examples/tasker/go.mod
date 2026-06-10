module github.com/go-fries/fries/queue/examples/tasker/v3

go 1.25.0

replace (
	github.com/go-fries/fries/codec/v3 => ../../../codec
	github.com/go-fries/fries/queue/adapter/memory/v3 => ../../adapter/memory
	github.com/go-fries/fries/queue/v3 => ../../
)

require (
	github.com/go-fries/fries/queue/adapter/memory/v3 v3.15.0-rc.1
	github.com/go-fries/fries/queue/v3 v3.15.0-rc.1
)

require github.com/go-fries/fries/codec/v3 v3.15.0-rc.1 // indirect
