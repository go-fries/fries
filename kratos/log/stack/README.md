# Stack Logger

Stack Logger is a logger for aggregating multiple loggers.

## Usage

```go
package main

import (
	"os"

	"github.com/go-fries/fries/kratos/log/stack/v3"
	"github.com/go-kratos/kratos/v2/log"
)

func main() {
	stackLogger := stack.New(
		log.NewStdLogger(os.Stdout),
		log.NewStdLogger(os.Stderr), // another logger
	)

	_ = stackLogger.Log(log.LevelInfo, "key", "value")
}
```