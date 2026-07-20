package response

// Body is the JSON response envelope.
//
// Status reports whether the application operation succeeded. Code is an
// optional application-defined code and does not represent the HTTP status
// code. Data is always encoded, using null when it is nil.
type Body struct {
	Status  bool   `json:"status"`
	Code    *int   `json:"code,omitempty"`
	Message string `json:"message"`
	Data    any    `json:"data"`
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
