// Package otel provides OpenTelemetry tracing middleware for Hyperf Jet clients.
//
// The middleware creates a client span for each Jet invocation and records
// semantic-convention attributes for JSON-RPC and the underlying HTTP
// transporter when those details are available from the Jet client context.
//
// Example:
//
//	transporter, err := jet.NewHTTPTransporter(
//		jet.WithHTTPTransporterAddr("https://api.example.com/rpc"),
//	)
//	if err != nil {
//		return err
//	}
//
//	client, err := jet.NewClient(
//		jet.WithService("example.UserService"),
//		jet.WithTransporter(transporter),
//		jet.WithMiddleware(otel.New(
//			otel.WithSchemaURL("https://opentelemetry.io/schemas/1.37.0"),
//			otel.WithAttributes(attribute.String("component", "jet")),
//		)),
//	)
//	if err != nil {
//		return err
//	}
//
//	var reply UserReply
//	return client.Invoke(ctx, "GetUser", UserRequest{ID: 1}, &reply)
package otel
