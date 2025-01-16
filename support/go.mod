module github.com/go-fries/fries/support/v3

go 1.22.10

replace (
	github.com/go-fries/fries/constraints/v3 => ../constraints
	github.com/go-fries/fries/errors/v3 => ../errors
)

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/go-fries/fries/constraints/v3 v3.0.0
	github.com/go-fries/fries/errors/v3 v3.0.0
	github.com/google/uuid v1.6.0
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/go-kratos/kratos/v2 v2.8.3 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250106144421-5f5ef82da422 // indirect
	google.golang.org/grpc v1.69.4 // indirect
	google.golang.org/protobuf v1.36.3 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
