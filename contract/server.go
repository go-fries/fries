package contract

import "context"

// Server is alias for github.com/go-kratos/kratos/v2/transport.Server
type Server interface {
	Start(context.Context) error
	Stop(context.Context) error
}
