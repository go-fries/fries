module github.com/go-fries/fries/filesystem/s3/v4

go 1.26.0

replace github.com/go-fries/fries/filesystem/v4 => ../

require (
	github.com/aws/aws-sdk-go-v2/service/s3 v1.113.2
	github.com/aws/smithy-go v1.28.2
	github.com/go-fries/fries/filesystem/v4 v4.2.0
	github.com/stretchr/testify v1.12.1
)

require (
	github.com/aws/aws-sdk-go-v2 v1.47.0 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.20 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.5.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.11.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.14.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.20.3 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
)
