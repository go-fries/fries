module github.com/go-fries/fries/coroutines/v3

go 1.23.0

replace (
	github.com/go-fries/fries/constraints/v3 => ./../constraints
	github.com/go-fries/fries/errors/v3 => ./../errors
	github.com/go-fries/fries/support/v3 => ./../support
)

require (
	github.com/go-fries/fries/support/v3 v3.7.0
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-fries/fries/errors/v3 v3.7.0 // indirect
	github.com/go-kratos/kratos/v2 v2.8.4 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250715232539-7130f93afb79 // indirect
	google.golang.org/grpc v1.74.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
