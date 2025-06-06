module github.com/go-fries/fries/examples/cloudevents/eventdispatcher/v3

go 1.23.0

replace github.com/go-fries/fries/cloudevents/eventdispatcher/v3 => ../../../cloudevents/eventdispatcher/

require (
	github.com/cloudevents/sdk-go/v2 v2.16.0
	github.com/go-fries/fries/cloudevents/eventdispatcher/v3 v3.4.0
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/sync v0.15.0 // indirect
)
