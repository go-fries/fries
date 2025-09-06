module github.com/go-fries/fries/codec/yaml/v3

go 1.24.0

replace github.com/go-fries/fries/codec/v3 => ../

require (
	github.com/go-fries/fries/codec/v3 v3.9.1
	github.com/stretchr/testify v1.11.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
)
