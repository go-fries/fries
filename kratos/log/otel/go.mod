module github.com/go-fries/fries/kratos/log/otel/v3

go 1.23.0

replace github.com/go-fries/fries/v3 => ../../../

require (
	github.com/go-fries/fries/v3 v3.7.1
	github.com/go-kratos/kratos/v2 v2.8.4
	github.com/stretchr/testify v1.10.0
	go.opentelemetry.io/otel/log v0.13.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/metric v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
