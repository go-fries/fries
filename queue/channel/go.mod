module github.com/go-fries/fries/queue/channel/v3

go 1.23.0

replace github.com/go-fries/fries/queue/v3 => ../

require github.com/go-fries/fries/queue/v3 v3.0.0-00010101000000-000000000000

require (
	github.com/go-fries/fries/codec/v3 v3.5.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
)
