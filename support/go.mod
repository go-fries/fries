module github.com/go-fries/fries/support/v4

go 1.25.0

replace github.com/go-fries/fries/errors/v4 => ../errors

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/go-fries/fries/errors/v4 v4.0.0
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/go-kratos/kratos/v2 v2.9.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260622175928-b703f567277d // indirect
	google.golang.org/grpc v1.81.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
