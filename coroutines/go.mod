module github.com/go-fries/fries/coroutines/v3

go 1.23.0

replace (
	github.com/go-fries/fries/errors/v3 => ../errors
	github.com/go-fries/fries/support/v3 => ./../support
)

require (
	github.com/go-fries/fries/support/v3 v3.0.0-alpha.2
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-fries/fries/errors/v3 v3.0.0-alpha.2 // indirect
	github.com/go-kratos/kratos/v2 v2.8.3 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250303144028-a0af3efb3deb // indirect
	google.golang.org/grpc v1.70.0 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
