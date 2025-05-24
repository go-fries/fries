package stack_test

import (
	"os"

	"github.com/go-fries/fries/kratos/log/stack/v3"
	"github.com/go-kratos/kratos/v2/log"
)

func Example_new() {
	stackLogger := stack.New(
		log.NewStdLogger(os.Stdout),
		log.NewStdLogger(os.Stderr), // another logger
	)

	_ = stackLogger.Log(log.LevelInfo, "key", "value")
}
