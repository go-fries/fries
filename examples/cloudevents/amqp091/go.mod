module github.com/go-fries/fries/examples/cloudevents/amqp091/v3

go 1.25.0

replace github.com/go-fries/fries/cloudevents/protocol/amqp091/v3 => ../../../cloudevents/protocol/amqp091/

require (
	github.com/cloudevents/sdk-go/v2 v2.16.2
	github.com/go-fries/fries/cloudevents/protocol/amqp091/v3 v3.12.0
	github.com/google/uuid v1.6.0
	github.com/rabbitmq/amqp091-go v1.10.0
)

require (
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.1 // indirect
)
