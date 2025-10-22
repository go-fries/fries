module github.com/go-fries/fries/event/example/v3

go 1.24.0

replace (
	github.com/go-fries/fries/event/middleware/recovery/v3 => ../middleware/recovery/
	github.com/go-fries/fries/event/v3 => ../
)

require (
	github.com/go-fries/fries/event/middleware/recovery/v3 v3.10.0
	github.com/go-fries/fries/event/v3 v3.10.0
)

require golang.org/x/sync v0.17.0 // indirect
