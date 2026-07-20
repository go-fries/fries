package response

import "context"

// ErrorMapper maps an application error to an HTTP status code and response
// body.
//
// Implementations should accept a nil error and return a successful response.
// They should also provide a safe fallback for unknown errors instead of
// exposing internal error messages.
type ErrorMapper interface {
	Map(context.Context, error) (httpStatus int, body Body)
}

// ErrorMapperFunc adapts a function to [ErrorMapper].
type ErrorMapperFunc func(context.Context, error) (httpStatus int, body Body)

// Map calls f with ctx and err.
//
// Map panics if f is nil.
func (f ErrorMapperFunc) Map(
	ctx context.Context,
	err error,
) (httpStatus int, body Body) {
	return f(ctx, err)
}
