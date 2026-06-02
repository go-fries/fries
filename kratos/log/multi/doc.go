// Package multi provides a Kratos logger that dispatches log calls to multiple
// logger implementations.
//
// Example:
//
//	import (
//		"os"
//
//		"github.com/go-fries/fries/kratos/log/multi/v3"
//		"github.com/go-kratos/kratos/v2/log"
//	)
//
//	logger := multi.New(
//		log.NewStdLogger(os.Stdout),
//		log.NewStdLogger(os.Stderr),
//	)
//
//	_ = logger.Log(log.LevelInfo, "key", "value")
package multi
