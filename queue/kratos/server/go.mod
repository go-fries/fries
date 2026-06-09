module github.com/go-fries/fries/queue/kratos/server/v3

go 1.25.0

replace (
	github.com/go-fries/fries/codec/v3 => ../../../codec
	github.com/go-fries/fries/queue/v3 => ../../
)

require (
	github.com/go-fries/fries/queue/v3 v3.14.0
	github.com/go-kratos/kratos/v2 v2.9.2
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-fries/fries/codec/v3 v3.14.0 // indirect
	github.com/go-playground/form/v4 v4.3.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
