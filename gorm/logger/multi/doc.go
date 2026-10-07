// Package multi provides a GORM logger that dispatches log calls to multiple
// logger implementations.
//
// Example:
//
//	import (
//		"github.com/go-fries/fries/gorm/logger/multi/v4"
//		gormotel "github.com/go-fries/fries/gorm/logger/otel/v4"
//		"go.opentelemetry.io/otel"
//		"gorm.io/gorm"
//		"gorm.io/gorm/logger"
//	)
//
//	db, err := gorm.Open(dialector, &gorm.Config{
//		Logger: multi.New(
//			logger.Default,
//			gormotel.New(gormotel.WithLoggerProvider(otel.GetLoggerProvider())),
//		),
//	})
package multi
