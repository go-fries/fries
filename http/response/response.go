package response

// Body is the JSON response envelope.
//
// Status reports whether the application operation succeeded. Code is an
// optional application-defined code and does not represent the HTTP status
// code. Data is omitted when its interface value is nil. A typed nil stored in
// Data is encoded as null.
type Body struct {
	Status  bool   `json:"status"`
	Code    *int   `json:"code,omitempty"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Success returns a successful response containing message and data.
func Success(message string, data any, options ...Option) Body {
	c := newConfig(data, options...)
	return Body{
		Status:  true,
		Code:    c.code,
		Message: message,
		Data:    c.data,
	}
}

// Failure returns a failed response containing message.
//
// Use [WithData] to include safe, structured failure details.
func Failure(message string, options ...Option) Body {
	c := newConfig(nil, options...)
	return Body{
		Status:  false,
		Code:    c.code,
		Message: message,
		Data:    c.data,
	}
}

// FromError returns a successful response when err is nil. Otherwise, it
// returns a failed response using err.Error() as the message.
//
// FromError does not infer an HTTP status code or application code. Only pass
// errors whose messages are safe to expose to API callers.
func FromError(err error, options ...Option) Body {
	if err == nil {
		return Success("", nil, options...)
	}
	return Failure(err.Error(), options...)
}
