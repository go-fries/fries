// Package otel provides a GORM logger implementation backed by the
// OpenTelemetry Logs API.
//
// Example:
//
//	import (
//		gormotel "github.com/go-fries/fries/gorm/logger/otel/v4"
//		"go.opentelemetry.io/otel"
//		"go.opentelemetry.io/otel/attribute"
//		"gorm.io/gorm"
//		"gorm.io/gorm/logger"
//	)
//
//	db, err := gorm.Open(dialector, &gorm.Config{
//		Logger: gormotel.New(
//			gormotel.WithLoggerProvider(otel.GetLoggerProvider()),
//			gormotel.WithLogLevel(logger.Warn),
//			gormotel.WithLogAttributes(attribute.String("component", "gorm")),
//		),
//	})
package otel
