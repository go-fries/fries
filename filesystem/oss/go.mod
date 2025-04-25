module github.com/go-fries/fries/filesystem/oss/v3

go 1.23.0

replace github.com/go-fries/fries/filesystem/v3 => ../

require (
	github.com/aliyun/alibabacloud-oss-go-sdk-v2 v1.2.2
	github.com/go-fries/fries/filesystem/v3 v3.0.2
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/time v0.11.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
