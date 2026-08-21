module github.com/go-fries/fries/filesystem/s3/v4

go 1.26.0

replace github.com/go-fries/fries/filesystem/v4 => ../

require (
	github.com/aws/aws-sdk-go-v2/service/s3 v1.107.2
	github.com/aws/smithy-go v1.27.8
	github.com/go-fries/fries/filesystem/v4 v4.0.0
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/aws/aws-sdk-go-v2 v1.43.6 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.18 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.37 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.37 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.38 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.30 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.37 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.38 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
