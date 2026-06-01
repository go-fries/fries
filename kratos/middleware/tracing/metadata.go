package tracing

import (
	"context"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/metadata"
	"go.opentelemetry.io/otel/propagation"
)

const serviceHeader = "x-md-service-name"

// Metadata propagates tracing-related Kratos metadata through text map carriers.
type Metadata struct{}

var _ propagation.TextMapPropagator = Metadata{}

// Inject sets metadata key-values from ctx into the carrier.
func (b Metadata) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	app, ok := kratos.FromContext(ctx)
	if ok {
		carrier.Set(serviceHeader, app.Name())
	}
}

// Extract adds metadata from the carrier to parent and returns the resulting context.
func (b Metadata) Extract(parent context.Context, carrier propagation.TextMapCarrier) context.Context {
	name := carrier.Get(serviceHeader)
	if name == "" {
		return parent
	}
	if md, ok := metadata.FromServerContext(parent); ok {
		md.Set(serviceHeader, name)
		return parent
	}
	md := metadata.New()
	md.Set(serviceHeader, name)
	return metadata.NewServerContext(parent, md)
}

// Fields returns the keys whose values are set with Inject.
func (b Metadata) Fields() []string {
	return []string{serviceHeader}
}
