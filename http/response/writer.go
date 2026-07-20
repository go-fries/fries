package response

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const contentType = "application/json; charset=utf-8"

// Write encodes body as JSON and writes it with httpStatus.
//
// Body is encoded before the response headers are committed. If encoding
// fails, Write returns an error without modifying w. Write panics if w is nil.
func Write(w http.ResponseWriter, httpStatus int, body Body) error {
	if w == nil {
		panic("response: nil response writer")
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("response: marshal body: %w", err)
	}
	payload = append(payload, '\n')

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(httpStatus)
	if _, err = w.Write(payload); err != nil {
		return fmt.Errorf("response: write body: %w", err)
	}
	return nil
}
