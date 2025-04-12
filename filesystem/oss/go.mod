module github.com/go-fries/fries/filesystem/oss/v3

go 1.23.0

replace github.com/go-fries/fries/filesystem/v3 => ../

require (
	github.com/aliyun/alibabacloud-oss-go-sdk-v2 v1.2.1
	github.com/go-fries/fries/filesystem/v3 v3.0.0
)

require golang.org/x/time v0.4.0 // indirect
