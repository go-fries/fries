module github.com/go-fries/fries/support/v3

go 1.25.0

replace github.com/go-fries/fries/errors/v3 => ../errors

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/go-fries/fries/errors/v3 v3.12.0
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/go-kratos/kratos/v2 v2.9.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260316180232-0b37fe3546d5 // indirect
	google.golang.org/grpc v1.79.2 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
