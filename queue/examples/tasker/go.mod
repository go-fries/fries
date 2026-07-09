module github.com/go-fries/fries/queue/examples/tasker/v4

go 1.25.0

replace (
	github.com/go-fries/fries/codec/v4 => ../../../codec
	github.com/go-fries/fries/queue/adapter/memory/v4 => ../../adapter/memory
	github.com/go-fries/fries/queue/v4 => ../../
)

require (
	github.com/go-fries/fries/queue/adapter/memory/v4 v4.0.0
	github.com/go-fries/fries/queue/v4 v4.0.0
)

require github.com/go-fries/fries/codec/v4 v4.0.0 // indirect
