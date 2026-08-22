module github.com/go-fries/fries/filesystem/oss/v4

go 1.26.0

replace github.com/go-fries/fries/filesystem/v4 => ../

require (
	github.com/aliyun/alibabacloud-oss-go-sdk-v2 v1.5.3
	github.com/go-fries/fries/filesystem/v4 v4.0.0
	github.com/stretchr/testify v1.12.1
)

require (
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/time v0.15.0 // indirect
)
