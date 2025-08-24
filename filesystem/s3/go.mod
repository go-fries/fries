module github.com/go-fries/fries/filesystem/s3/v3

go 1.24.0

replace github.com/go-fries/fries/filesystem/v3 => ../

require (
	github.com/aws/aws-sdk-go-v2/service/s3 v1.87.1
	github.com/aws/smithy-go v1.22.5
	github.com/go-fries/fries/filesystem/v3 v3.8.0
)

require (
	github.com/aws/aws-sdk-go-v2 v1.38.1 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.4 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.4 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.8.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.4 // indirect
)
