// Package otel provides a Kratos logger implementation backed by the
// OpenTelemetry Logs API.
//
// Example:
//
//	import (
//		kratoslog "github.com/go-kratos/kratos/v2/log"
//		otelkratos "github.com/go-fries/fries/kratos/log/otel/v3"
//		"go.opentelemetry.io/otel/attribute"
//		"go.opentelemetry.io/otel/log/global"
//	)
//
//	logger := otelkratos.NewLogger(
//		otelkratos.WithLoggerProvider(global.GetLoggerProvider()),
//		otelkratos.WithSchemaURL("https://opentelemetry.io/schemas/1.37.0"),
//		otelkratos.WithAttributes(attribute.String("service.name", "example")),
//	)
//
//	helper := kratoslog.NewHelper(logger)
//	helper.Infow("msg", "server started")
package otel
