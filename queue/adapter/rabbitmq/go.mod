module github.com/go-fries/fries/queue/adapter/rabbitmq/v3

go 1.25.0

replace github.com/go-fries/fries/queue/v3 => ../../

replace github.com/go-fries/fries/codec/v3 => ../../../codec

require (
	github.com/go-fries/fries/queue/v3 v3.15.0-rc.1
	github.com/rabbitmq/amqp091-go v1.11.0
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-fries/fries/codec/v3 v3.15.0-rc.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
