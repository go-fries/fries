// Package multi provides a GORM logger that dispatches log calls to multiple
// logger implementations.
//
// Example:
//
//	import (
//		"github.com/go-fries/fries/gorm/logger/multi/v3"
//		"github.com/go-fries/fries/gorm/logger/otel/v3"
//		"go.opentelemetry.io/otel/log/global"
//		"gorm.io/gorm"
//		"gorm.io/gorm/logger"
//	)
//
//	db, err := gorm.Open(dialector, &gorm.Config{
//		Logger: multi.New(
//			logger.Default,
//			otel.New(otel.WithLoggerProvider(global.GetLoggerProvider())),
//		),
//	})
package multi
