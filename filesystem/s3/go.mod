module github.com/go-fries/fries/filesystem/s3/v4

go 1.26.0

replace github.com/go-fries/fries/filesystem/v4 => ../

require (
	github.com/aws/aws-sdk-go-v2/service/s3 v1.116.2
	github.com/aws/smithy-go v1.28.11
	github.com/go-fries/fries/filesystem/v4 v4.3.0
	github.com/stretchr/testify v1.12.1
)

require (
	github.com/aws/aws-sdk-go-v2 v1.48.2 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.14 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.14 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.5.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.27 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.11.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.14.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.20.14 // indirect
	go.yaml.in/yaml/v3 v3.0.6 // indirect
)
