module github.com/go-fries/fries/codec/proto/v4

go 1.26.0

replace github.com/go-fries/fries/codec/v4 => ../

require (
	github.com/go-fries/fries/codec/v4 v4.0.0
	github.com/stretchr/testify v1.12.1
	google.golang.org/protobuf v1.36.12
)

require go.yaml.in/yaml/v3 v3.0.5 // indirect
