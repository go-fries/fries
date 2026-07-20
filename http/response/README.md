# HTTP Response

`response` provides a small JSON envelope for HTTP APIs without binding
applications to a router or web framework.

## Installation

```bash
go get github.com/go-fries/fries/http/response/v4
```

## Response format

Every response includes an application status, a public message, and data. The
application code is optional:

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
automatically copied from the HTTP status code. When there is no data, `data`
is encoded as `null`.

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

