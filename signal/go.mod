module github.com/go-fries/fries/signal/v3

go 1.23.0

replace github.com/go-fries/fries/contract/v3 => ../contract

require (
	github.com/go-fries/fries/contract/v3 v3.0.0-rc.1
	github.com/go-kratos/kratos/v2 v2.8.4
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sync v0.12.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
