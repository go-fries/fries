# HTTP Response

`response` provides a small JSON envelope for HTTP APIs without binding
applications to a router or web framework.

## Installation

```bash
go get github.com/go-fries/fries/http/response/v4
```

## Response format

Every response includes an application status and a public message. The
application code and data are optional:

```json
{
  "status": true,
  "code": 10000,
  "message": "Scratch 11 is working properly.",
  "data": {
    "id": 11,
    "name": "Scratch 11"
  }
}
```

`status` reports whether the application operation succeeded. It does not
replace the HTTP status code. `code` is application-defined and is not
automatically copied from the HTTP status code.

`data` is omitted when its interface value is nil. Empty non-nil values,
including slices, maps, strings, and numeric zero, remain in the response. A
typed nil pointer, slice, or map stored in `data` is encoded as `null` because
the containing interface is non-nil.

## Write a response

Use `Success` or `Failure` to build a body, then pass the real HTTP status code
to `Write`:

```go
func showScratch(w http.ResponseWriter, _ *http.Request) {
	scratch := struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}{
		ID:   11,
		Name: "Scratch 11",
	}

	body := response.Success(
		"Scratch 11 is working properly.",
		scratch,
		response.WithCode(10000),
	)

	if err := response.Write(w, http.StatusOK, body); err != nil {
		slog.Error("write response", "error", err)
	}
}
```

Failures use the same envelope:

```go
body := response.Failure(
	"Scratch does not exist.",
	response.WithCode(10404),
)

if err := response.Write(w, http.StatusNotFound, body); err != nil {
	slog.Error("write response", "error", err)
}
```

For errors that are already safe to expose, `FromError` converts a nil error
to a successful response and a non-nil error to a failed response:

```go
body := response.FromError(
	err,
	response.WithCode(10422),
)
```

`FromError` uses `err.Error()` as the failure message. Do not pass database,
network, credential, or other internal errors to it. Log those errors
separately and use `Failure` with a public message instead.

## Map application errors

Use `ErrorMapper` to centralize the conversion from domain errors to HTTP
status codes and safe response bodies:

```go
type applicationErrorMapper struct{}

func (applicationErrorMapper) Map(
	_ context.Context,
	err error,
) (int, response.Body) {
	switch {
	case err == nil:
		return http.StatusOK, response.FromError(nil)

	case errors.Is(err, repository.ErrNotFound):
		return http.StatusNotFound, response.Failure(
			"Resource not found.",
			response.WithCode(10404),
		)

	default:
		return http.StatusInternalServerError, response.Failure(
			"Unable to process the request.",
			response.WithCode(10500),
		)
	}
}
```

Handlers can use the mapper without duplicating error rules:

```go
httpStatus, body := mapper.Map(r.Context(), err)
if err := response.Write(w, httpStatus, body); err != nil {
	slog.Error("write response", "error", err)
}
```

`ErrorMapperFunc` adapts a function when a dedicated mapper type is
unnecessary. Mappers should accept a nil error and return a successful
response. Unknown errors should always use a safe fallback message rather than
exposing `err.Error()`.

Use `WithData` when a failure needs safe, structured details:

```go
body := response.Failure(
	"The request contains invalid fields.",
	response.WithCode(10422),
	response.WithData(map[string][]string{
		"name": {"name is required"},
	}),
)
```

Do not place internal errors, credentials, connection details, or stack traces
in `message` or `data`. Log internal errors separately and return a message
that is safe for API callers.

`Write` serializes the complete body before committing the response headers.
If serialization fails, it returns an error without sending a partial JSON
response.

## Frameworks

`Body` is an ordinary Go value and can be passed directly to framework JSON
helpers:

```go
c.JSON(
	http.StatusOK,
	response.Success("OK", data),
)
```

Framework-specific adapters are not required.
