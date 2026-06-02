// Package otel provides a GORM logger implementation backed by the
// OpenTelemetry Logs API.
//
// Example:
//
//	import (
//		gormotel "github.com/go-fries/fries/gorm/logger/otel/v3"
//		"go.opentelemetry.io/otel/log/global"
//		"gorm.io/gorm"
//		"gorm.io/gorm/logger"
//	)
//
//	db, err := gorm.Open(dialector, &gorm.Config{
//		Logger: gormotel.New(
//			gormotel.WithLoggerProvider(global.GetLoggerProvider()),
//			gormotel.WithLogLevel(logger.Warn),
//		),
//	})
package otel
