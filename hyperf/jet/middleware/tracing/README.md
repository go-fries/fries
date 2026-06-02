# Tracing - Hyperf Jet middleware

OpenTelemetry tracing middleware for Hyperf Jet clients.

The middleware creates a client span for each Jet invocation. When the Jet
client is available in the context, spans include JSON-RPC attributes and HTTP
transporter attributes such as `http.request.method`, `url.full`,
`server.address`, and `server.port`.

Trace context propagation is not injected by this middleware because the current
Jet middleware layer does not control HTTP request headers. Propagation should be
implemented at the transporter layer when header injection is needed.

## Usage

```go
package main

import (
	"context"

	"github.com/go-fries/fries/hyperf/jet/middleware/tracing/v3"
	"github.com/go-fries/fries/hyperf/jet/v3"
	"go.opentelemetry.io/otel/attribute"
)

func main() {
	transporter, err := jet.NewHTTPTransporter(
		jet.WithHTTPTransporterAddr("https://api.example.com/rpc"),
	)
	if err != nil {
		panic(err)
	}

	client, err := jet.NewClient(
		jet.WithService("example.UserService"),
		jet.WithTransporter(transporter),
		jet.WithMiddleware(tracing.New(
			tracing.WithVersion(tracing.Version()),
			tracing.WithSchemaURL("https://opentelemetry.io/schemas/1.37.0"),
			tracing.WithAttributes(attribute.String("component", "jet")),
		)),
	)
	if err != nil {
		panic(err)
	}

	var reply UserReply
	if err := client.Invoke(context.Background(), "GetUser", UserRequest{ID: 1}, &reply); err != nil {
		panic(err)
	}
}

type UserRequest struct {
	ID int
}

type UserReply struct {
	Name string
}
```
