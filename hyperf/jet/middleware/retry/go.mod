module github.com/go-fries/fries/hyperf/jet/middleware/retry/v4

go 1.25.0

replace (
	github.com/go-fries/fries/hyperf/jet/middleware/timeout/v4 => ../timeout
	github.com/go-fries/fries/hyperf/jet/v4 => ../../
)

require (
	github.com/go-fries/fries/hyperf/jet/middleware/timeout/v4 v4.0.0
	github.com/go-fries/fries/hyperf/jet/v4 v4.0.0
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
