module github.com/go-fries/fries/hyperf/jet/middleware/retry/v3

go 1.24.0

replace (
	github.com/go-fries/fries/hyperf/jet/middleware/timeout/v3 => ../timeout
	github.com/go-fries/fries/hyperf/jet/v3 => ../../
)

require (
	github.com/go-fries/fries/hyperf/jet/middleware/timeout/v3 v3.11.0
	github.com/go-fries/fries/hyperf/jet/v3 v3.11.0
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
