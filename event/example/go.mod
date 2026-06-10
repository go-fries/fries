module github.com/go-fries/fries/event/example/v3

go 1.25.0

replace (
	github.com/go-fries/fries/event/middleware/recovery/v3 => ../middleware/recovery/
	github.com/go-fries/fries/event/v3 => ../
)

require (
	github.com/go-fries/fries/event/middleware/recovery/v3 v3.15.0-rc.1
	github.com/go-fries/fries/event/v3 v3.15.0-rc.1
)

require golang.org/x/sync v0.21.0 // indirect
