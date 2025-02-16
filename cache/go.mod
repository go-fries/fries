module github.com/go-fries/fries/cache/v3

go 1.22.10

replace github.com/go-fries/fries/locker/v3 => ../locker

require (
	github.com/go-fries/fries/locker/v3 v3.0.0-alpha.1
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
