module github.com/go-fries/fries/event/example/v4

go 1.25.0

replace (
	github.com/go-fries/fries/event/middleware/recovery/v4 => ../middleware/recovery/
	github.com/go-fries/fries/event/v4 => ../
)

require (
	github.com/go-fries/fries/event/middleware/recovery/v4 v4.0.0
	github.com/go-fries/fries/event/v4 v4.0.0
)

require golang.org/x/sync v0.21.0 // indirect
