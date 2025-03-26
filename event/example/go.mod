module github.com/go-fries/fries/event/example/v3

go 1.23.0

replace (
	github.com/go-fries/fries/event/middleware/recovery/v3 => ../middleware/recovery/
	github.com/go-fries/fries/event/v3 => ../
)

require (
	github.com/go-fries/fries/event/middleware/recovery/v3 v3.0.0-rc.1
	github.com/go-fries/fries/event/v3 v3.0.0-rc.1
)

require golang.org/x/sync v0.12.0 // indirect
