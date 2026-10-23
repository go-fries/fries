module github.com/go-fries/fries/filesystem/s3/v4

go 1.26.0

replace github.com/go-fries/fries/filesystem/v4 => ../

require (
	github.com/aws/aws-sdk-go-v2/service/s3 v1.114.5
	github.com/aws/smithy-go v1.28.6
	github.com/go-fries/fries/filesystem/v4 v4.3.0
	github.com/stretchr/testify v1.12.1
)

require (
	github.com/aws/aws-sdk-go-v2 v1.47.4 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.24 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.7 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.7 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.5.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.22 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.11.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.14.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.20.7 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
)
