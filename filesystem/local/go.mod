module github.com/go-fries/fries/filesystem/local/v3

go 1.24.0

replace github.com/go-fries/fries/filesystem/v3 => ../

require (
	github.com/go-fries/fries/filesystem/v3 v3.12.0
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
