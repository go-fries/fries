module github.com/go-fries/fries/webhook/sender/v4

go 1.26.0

require (
	github.com/go-fries/fries/webhook/v4 v4.0.0
	github.com/stretchr/testify v1.12.1
)

require go.yaml.in/yaml/v3 v3.0.5 // indirect

replace github.com/go-fries/fries/webhook/v4 => ../
