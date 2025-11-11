module github.com/go-fries/fries/crontab/example/v3

go 1.24.0

replace github.com/go-fries/fries/crontab/v3 => ../

require (
	github.com/flc1125/go-cron/v4 v4.7.0
	github.com/go-fries/fries/crontab/v3 v3.10.0
	github.com/go-kratos/kratos/v2 v2.9.1
)

require (
	github.com/go-playground/form/v4 v4.3.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/sync v0.18.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251110190251-83f479183930 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251110190251-83f479183930 // indirect
	google.golang.org/grpc v1.76.0 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
