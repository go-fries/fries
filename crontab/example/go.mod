module github.com/go-fries/fries/crontab/example/v3

go 1.22.10

replace github.com/go-fries/fries/crontab/v3 => ../

require (
	github.com/flc1125/go-cron/v4 v4.3.2
	github.com/go-fries/fries/crontab/v3 v3.0.0
	github.com/go-kratos/kratos/v2 v2.8.3
)

require (
	github.com/go-playground/form/v4 v4.2.1 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250122153221-138b5a5a4fd4 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250115164207-1a7da9e5054f // indirect
	google.golang.org/grpc v1.69.4 // indirect
	google.golang.org/protobuf v1.36.3 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
