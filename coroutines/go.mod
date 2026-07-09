module github.com/go-fries/fries/coroutines/v4

go 1.25.0

replace (
	github.com/go-fries/fries/constraints/v4 => ./../constraints
	github.com/go-fries/fries/errors/v4 => ./../errors
	github.com/go-fries/fries/support/v4 => ./../support
)

require (
	github.com/go-fries/fries/support/v4 v4.0.0-beta.1
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-fries/fries/errors/v4 v4.0.0-beta.1 // indirect
	github.com/go-kratos/kratos/v3 v3.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260706201446-f0a921348800 // indirect
	google.golang.org/grpc v1.82.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
