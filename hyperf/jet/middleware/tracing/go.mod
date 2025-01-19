module github.com/go-fries/fries/hyperf/jet/middleware/tracing/v3

go 1.22.10

replace github.com/go-fries/fries/hyperf/jet/v3 => ../../

require (
	github.com/go-fries/fries/hyperf/jet/v3 v3.0.0
	go.opentelemetry.io/otel v1.34.0
	go.opentelemetry.io/otel/trace v1.34.0
)

require (
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/metric v1.34.0 // indirect
	golang.org/x/text v0.21.0 // indirect
)
