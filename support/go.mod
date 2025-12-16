module github.com/go-fries/fries/support/v3

go 1.24.0

replace github.com/go-fries/fries/errors/v3 => ../errors

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/go-fries/fries/errors/v3 v3.11.0
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/go-kratos/kratos/v2 v2.9.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251213004720-97cd9d5aeac2 // indirect
	google.golang.org/grpc v1.77.0 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
